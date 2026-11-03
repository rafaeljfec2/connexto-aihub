package resources

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Service owns policy administration and admission decisions.
type Service struct {
	repo Repository
	now  func() time.Time
}

// Option configures the service.
type Option func(*Service)

// WithClock overrides the UTC clock used for admission periods.
func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		s.now = now
	}
}

// NewService builds the resource service.
func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{repo: repo, now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CreatePolicy stores a new scope policy.
func (s *Service) CreatePolicy(ctx context.Context, input NewPolicyInput) (ResourcePolicy, error) {
	if err := validateScope(input.Scope); err != nil {
		return ResourcePolicy{}, err
	}
	if input.DailyLimit <= 0 {
		return ResourcePolicy{}, fmt.Errorf("%w: daily_limit must be positive", ErrInvalidInput)
	}
	return s.repo.CreatePolicy(ctx, input)
}

// UpdatePolicy changes the daily limit of an existing policy.
func (s *Service) UpdatePolicy(ctx context.Context, id int64, input UpdatePolicyInput) (ResourcePolicy, error) {
	if input.DailyLimit != nil && *input.DailyLimit <= 0 {
		return ResourcePolicy{}, fmt.Errorf("%w: daily_limit must be positive", ErrInvalidInput)
	}
	return s.repo.UpdatePolicy(ctx, id, input)
}

// SetPolicyEnabled toggles whether a policy participates in admission.
func (s *Service) SetPolicyEnabled(ctx context.Context, id int64, enabled bool) (ResourcePolicy, error) {
	return s.repo.SetPolicyEnabled(ctx, id, enabled)
}

// GetPolicy returns one policy.
func (s *Service) GetPolicy(ctx context.Context, id int64) (ResourcePolicy, error) {
	return s.repo.GetPolicy(ctx, id)
}

// ListPolicies returns every policy ordered by scope key.
func (s *Service) ListPolicies(ctx context.Context) ([]ResourcePolicy, error) {
	return s.repo.ListPolicies(ctx)
}

// Admit reserves one admission unit for the routed target. It assigns a stable
// internal identifier when request.ID is empty.
func (s *Service) Admit(ctx context.Context, request AdmissionRequest) (AdmissionDecision, error) {
	request.ProviderID = strings.TrimSpace(request.ProviderID)
	request.ModelID = strings.TrimSpace(request.ModelID)
	if request.ProviderID == "" || request.ModelID == "" {
		return AdmissionDecision{}, fmt.Errorf("%w: provider_id and model_id are required", ErrInvalidInput)
	}

	if request.ID == "" {
		id, err := NewAdmissionID()
		if err != nil {
			return AdmissionDecision{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		request.ID = id
	} else if strings.TrimSpace(request.ID) == "" {
		return AdmissionDecision{}, fmt.Errorf("%w: admission id must not be blank", ErrInvalidInput)
	}

	decision, err := s.repo.Admit(ctx, request, s.now().UTC())
	if err != nil {
		return AdmissionDecision{}, err
	}
	if !decision.Allowed {
		return decision, ErrLimitExceeded
	}
	return decision, nil
}

// NewAdmissionID returns a stable internal admission identifier (ADR-023).
func NewAdmissionID() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate admission id: %w", err)
	}
	return "aihub-adm-" + hex.EncodeToString(random[:]), nil
}

func validateScope(scope ResourceScope) error {
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
