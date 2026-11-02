package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// BootstrapInput seeds a configuration from process settings when absent.
type BootstrapInput struct {
	ID                     string
	Name                   string
	Type                   string
	Endpoint               string
	Credential             string
	Enabled                bool
	WarnIgnoredEnvironment bool
}

// Bootstrap inserts missing settings, preserves stored values, and applies the
// persisted configuration to the adapter bound to that provider ID.
func (s *Service) Bootstrap(ctx context.Context, input BootstrapInput) (Record, error) {
	if err := s.validateIdentity(input.ID, input.Name, input.Type); err != nil {
		return Record{}, err
	}
	if _, err := s.repo.Get(ctx, input.ID); errors.Is(err, ErrNotFound) {
		if err := s.validateEndpoint(strings.TrimSpace(input.Endpoint)); err != nil {
			return Record{}, err
		}
	} else if err != nil {
		return Record{}, err
	}
	sealed, err := s.sealOptional(input.ID, input.Credential)
	if err != nil {
		return Record{}, err
	}
	record, err := s.repo.EnsureConfigured(ctx, NewRecord{
		ID: input.ID, Name: strings.TrimSpace(input.Name), Type: input.Type,
		Endpoint: strings.TrimSpace(input.Endpoint), Enabled: input.Enabled,
		Credential: sealed,
	})
	if err != nil {
		return Record{}, err
	}
	if err := s.validateEndpoint(record.Endpoint); err != nil {
		return Record{}, fmt.Errorf("persisted provider %s: %w", record.ID, err)
	}
	return record, s.apply(ctx, record)
}

// Apply loads and applies one persisted provider configuration.
func (s *Service) Apply(ctx context.Context, id string) error {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.apply(ctx, record)
}
