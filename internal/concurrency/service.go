package concurrency

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const DefaultLeaseTTL = 5 * time.Minute
const DefaultOperationTimeout = 2 * time.Second

const releaseMaxAttempts = 5
const releaseInitialBackoff = 50 * time.Millisecond

// Service owns concurrency policy administration and slot lifecycle.
type Service struct {
	repo             Repository
	logger           *slog.Logger
	now              func() time.Time
	leaseTTL         time.Duration
	operationTimeout time.Duration
}

// Option configures the service.
type Option func(*Service)

// WithClock overrides UTC clock.
func WithClock(now func() time.Time) Option {
	return func(s *Service) { s.now = now }
}

// WithLeaseTTL overrides lease duration for acquire/renew.
func WithLeaseTTL(ttl time.Duration) Option {
	return func(s *Service) {
		if ttl > 0 {
			s.leaseTTL = ttl
		}
	}
}

// WithOperationTimeout bounds lease persistence operations and release retries.
func WithOperationTimeout(timeout time.Duration) Option {
	return func(s *Service) {
		if timeout > 0 {
			s.operationTimeout = timeout
		}
	}
}

// WithLogger attaches structured logging for concurrency events.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) { s.logger = logger }
}

// NewService builds the concurrency service.
func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{
		repo: repo, now: time.Now, leaseTTL: DefaultLeaseTTL,
		operationTimeout: DefaultOperationTimeout,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) CreatePolicy(ctx context.Context, input NewPolicyInput) (Policy, error) {
	if err := validateScope(input.Scope); err != nil {
		return Policy{}, err
	}
	if input.ConcurrencyLimit <= 0 {
		return Policy{}, fmt.Errorf("%w: concurrency_limit must be positive", ErrInvalidInput)
	}
	return s.repo.CreatePolicy(ctx, input)
}

func (s *Service) UpdatePolicy(ctx context.Context, id int64, input UpdatePolicyInput) (Policy, error) {
	if input.ConcurrencyLimit != nil && *input.ConcurrencyLimit <= 0 {
		return Policy{}, fmt.Errorf("%w: concurrency_limit must be positive", ErrInvalidInput)
	}
	return s.repo.UpdatePolicy(ctx, id, input)
}

func (s *Service) SetPolicyEnabled(ctx context.Context, id int64, enabled bool) (Policy, error) {
	return s.repo.SetPolicyEnabled(ctx, id, enabled)
}

func (s *Service) ListPolicies(ctx context.Context) ([]Policy, error) {
	return s.repo.ListPolicies(ctx)
}

func (s *Service) GetPolicyByScope(ctx context.Context, scope Scope) (Policy, error) {
	if err := validateScope(scope); err != nil {
		return Policy{}, err
	}
	return s.repo.GetPolicyByScope(ctx, scope)
}

func (s *Service) UpsertPolicy(ctx context.Context, input UpsertPolicyInput) (Policy, bool, error) {
	if err := validateScope(input.Scope); err != nil {
		return Policy{}, false, err
	}
	if input.ConcurrencyLimit <= 0 {
		return Policy{}, false, fmt.Errorf("%w: concurrency_limit must be positive", ErrInvalidInput)
	}
	existing, err := s.repo.GetPolicyByScope(ctx, input.Scope)
	if errors.Is(err, ErrNotFound) {
		policy, createErr := s.repo.CreatePolicy(ctx, NewPolicyInput{
			Scope: input.Scope, ConcurrencyLimit: input.ConcurrencyLimit, Enabled: input.Enabled,
		})
		return policy, true, createErr
	}
	if err != nil {
		return Policy{}, false, err
	}
	policy, err := s.repo.ConfigurePolicy(ctx, existing.ID, input.ConcurrencyLimit, input.Enabled)
	return policy, false, err
}

func (s *Service) ListUsage(ctx context.Context, query UsageQuery) ([]UsageSnapshot, error) {
	query.ProviderID = strings.TrimSpace(query.ProviderID)
	query.ModelID = strings.TrimSpace(query.ModelID)
	return s.repo.ListUsage(ctx, query)
}

// Acquire reserves slots for one execution. Idempotent for the same execution ID.
func (s *Service) Acquire(ctx context.Context, request AcquireRequest) (AcquireDecision, error) {
	request.ExecutionID = strings.TrimSpace(request.ExecutionID)
	request.ProviderID = strings.TrimSpace(request.ProviderID)
	request.ModelID = strings.TrimSpace(request.ModelID)
	if request.ExecutionID == "" || request.ProviderID == "" || request.ModelID == "" {
		return AcquireDecision{}, fmt.Errorf("%w: execution_id, provider_id and model_id are required", ErrInvalidInput)
	}
	now := s.now().UTC()
	decision, err := s.repo.Acquire(ctx, request, now, s.leaseTTL)
	if err != nil {
		s.logEvent("concurrency_admission_failed", request, decision, err)
		return AcquireDecision{}, err
	}
	for _, lease := range decision.ExpiredLeases {
		s.logSimple("concurrency_lease_expired",
			"execution_id", lease.ExecutionID,
			"provider_id", lease.ProviderID,
			"model_id", lease.ModelID)
	}
	if !decision.Allowed {
		s.logEvent("concurrency_rejected", request, decision, ErrLimitExceeded)
		return decision, ErrLimitExceeded
	}
	s.logEvent("concurrency_acquired", request, decision, nil)
	return decision, nil
}

// Release frees a slot. Idempotent. Retries transient storage failures.
func (s *Service) Release(ctx context.Context, executionID string) error {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return fmt.Errorf("%w: execution_id is required", ErrInvalidInput)
	}
	operationContext, cancel := context.WithTimeout(ctx, s.operationTimeout)
	defer cancel()
	delay := releaseInitialBackoff
	var lastErr error
	for attempt := 1; attempt <= releaseMaxAttempts; attempt++ {
		lastErr = s.repo.Release(operationContext, executionID)
		if lastErr == nil {
			s.logSimple("concurrency_released", "execution_id", executionID)
			return nil
		}
		if !errors.Is(lastErr, ErrUnavailable) || attempt == releaseMaxAttempts {
			break
		}
		select {
		case <-operationContext.Done():
			lastErr = operationContext.Err()
			attempt = releaseMaxAttempts
		case <-time.After(delay):
			delay *= 2
		}
	}
	s.logSimple("concurrency_admission_failed", "execution_id", executionID, "phase", "release", "error", lastErr.Error())
	return lastErr
}

// RenewLease extends an active lease while execution continues.
func (s *Service) RenewLease(ctx context.Context, executionID string) error {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return fmt.Errorf("%w: execution_id is required", ErrInvalidInput)
	}
	operationContext, cancel := context.WithTimeout(ctx, s.operationTimeout)
	defer cancel()
	now := s.now().UTC()
	if err := s.repo.RenewLease(operationContext, executionID, now, s.leaseTTL); err != nil {
		return err
	}
	s.logSimple("concurrency_lease_renewed", "execution_id", executionID)
	return nil
}

func validateScope(scope Scope) error {
	switch scope.Kind {
	case ScopeGlobal:
		if scope.ProviderID != "" || scope.ModelID != "" {
			return fmt.Errorf("%w: global scope must not include provider or model", ErrInvalidInput)
		}
	case ScopeProvider:
		if strings.TrimSpace(scope.ProviderID) == "" || scope.ModelID != "" {
			return fmt.Errorf("%w: provider scope requires provider_id only", ErrInvalidInput)
		}
	case ScopeModel:
		if strings.TrimSpace(scope.ProviderID) == "" || strings.TrimSpace(scope.ModelID) == "" {
			return fmt.Errorf("%w: model scope requires provider_id and model_id", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unknown scope %q", ErrInvalidInput, scope.Kind)
	}
	return nil
}

func (s *Service) logEvent(name string, request AcquireRequest, decision AcquireDecision, err error) {
	if s.logger == nil {
		return
	}
	attrs := []any{
		"event", name,
		"execution_id", request.ExecutionID,
		"provider_id", request.ProviderID,
		"model_id", request.ModelID,
	}
	if decision.Reason != "" {
		attrs = append(attrs, "reason", decision.Reason)
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	s.logger.InfoContext(context.Background(), name, attrs...)
}

func (s *Service) logSimple(name string, attrs ...any) {
	if s.logger == nil {
		return
	}
	pairs := append([]any{"event", name}, attrs...)
	s.logger.InfoContext(context.Background(), name, pairs...)
}
