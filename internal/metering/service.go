package metering

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
)

// ErrInvalidInput marks malformed persistence input.
var ErrInvalidInput = errors.New("metering: invalid input")

// Repository persists normalized usage rows.
type Repository interface {
	Insert(ctx context.Context, record Record) (inserted bool, err error)
}

// Service normalizes and persists provider usage without affecting execution outcomes.
type Service struct {
	repo   Repository
	logger *slog.Logger
	clock  func() time.Time
}

// Option configures the metering service.
type Option func(*Service)

// WithLogger installs structured logging for persistence failures.
func WithLogger(logger *slog.Logger) Option {
	return func(service *Service) {
		service.logger = logger
	}
}

// WithClock overrides time for tests.
func WithClock(clock func() time.Time) Option {
	return func(service *Service) {
		service.clock = clock
	}
}

// NewService builds a metering service. A nil repository disables persistence.
func NewService(repo Repository, options ...Option) *Service {
	service := &Service{repo: repo, clock: time.Now}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

// RecordFromCapture normalizes capture output and persists it once per execution.
func (s *Service) RecordFromCapture(ctx context.Context, input RecordInput) error {
	if s == nil || s.repo == nil {
		return nil
	}
	record, err := s.buildRecord(input)
	if err != nil {
		return err
	}
	inserted, err := s.repo.Insert(ctx, record)
	if err != nil {
		s.logFailure(record.ExecutionID, record.ProviderID, err)
		return err
	}
	if !inserted {
		return nil
	}
	return nil
}

// RecordFromCaptureBestEffort persists usage and logs failures without returning them to callers.
func (s *Service) RecordFromCaptureBestEffort(ctx context.Context, input RecordInput) {
	if err := s.RecordFromCapture(ctx, input); err != nil {
		return
	}
}

func (s *Service) buildRecord(input RecordInput) (Record, error) {
	executionID := strings.TrimSpace(input.ExecutionID)
	providerID := strings.TrimSpace(input.ProviderID)
	modelID := strings.TrimSpace(input.ModelID)
	if executionID == "" || providerID == "" || modelID == "" {
		return Record{}, ErrInvalidInput
	}
	normalized := NormalizeExecutionUsage(providerID, input.Capture)
	normalized.ExecutionID = executionID
	normalized.ProviderID = providerID
	normalized.ModelID = modelID
	if normalized.RecordedAt.IsZero() {
		normalized.RecordedAt = s.clock().UTC()
	}
	return normalized, nil
}

func (s *Service) logFailure(executionID, providerID string, err error) {
	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Error("provider usage persistence failed",
		"class", "provider_usage_persist_failed",
		"execution_id", executionID,
		"provider_id", providerID,
		"error", err.Error(),
	)
}
