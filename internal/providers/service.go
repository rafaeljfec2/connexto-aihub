package providers

import (
	"context"
	"fmt"
	"strings"
)

// ProbeFunc checks that an upstream accepts the endpoint and credential.
type ProbeFunc func(ctx context.Context, endpoint, credential string) error

// CreateInput is the administrative input for a new configuration.
type CreateInput struct {
	ID         string
	Name       string
	Type       string
	Endpoint   string
	Credential string
	Enabled    bool
}

// UpdateInput changes the optional fields of a configuration. A nil field is
// unchanged; an empty Credential is rejected rather than clearing the stored
// secret silently.
type UpdateInput struct {
	Name       *string
	Endpoint   *string
	Credential *string
}

// Service owns provider administration: validation, credential protection,
// persistence and propagation of the configuration to the bound runtime.
type Service struct {
	repo         Repository
	cipher       *CredentialCipher
	registry     *Registry
	probes       map[string]ProbeFunc
	allowedHosts map[string]struct{}
}

// NewService builds the administrative service. allowedHosts is the exact
// host allowlist for provider endpoints (AIHUB_PROVIDER_ALLOWED_HOSTS): the
// credential is only ever sent to these hosts.
func NewService(repo Repository, cipher *CredentialCipher, registry *Registry, probe ProbeFunc, allowedHosts []string) *Service {
	return NewServiceWithProbes(repo, cipher, registry, map[string]ProbeFunc{TypeOpenCodeGo: probe}, allowedHosts)
}

// Create stores a new configuration. It does not bind a runtime: only the
// startup bootstrap and the update paths reconfigure an adapter.
func (s *Service) Create(ctx context.Context, input CreateInput) (Record, error) {
	if err := s.validateIdentity(input.ID, input.Name, input.Type); err != nil {
		return Record{}, err
	}

	if err := s.validateEndpoint(strings.TrimSpace(input.Endpoint)); err != nil {
		return Record{}, err
	}

	sealed, err := s.sealOptional(input.ID, input.Credential)
	if err != nil {
		return Record{}, err
	}

	return s.repo.Create(ctx, NewRecord{
		ID:         input.ID,
		Name:       strings.TrimSpace(input.Name),
		Type:       input.Type,
		Endpoint:   strings.TrimSpace(input.Endpoint),
		Enabled:    input.Enabled,
		Credential: sealed,
	})
}

// Get returns one configuration.
func (s *Service) Get(ctx context.Context, id string) (Record, error) {
	return s.repo.Get(ctx, id)
}

// List returns every configuration ordered by id.
func (s *Service) List(ctx context.Context) ([]Record, error) {
	return s.repo.List(ctx)
}

// Update changes name, endpoint and/or credential, then propagates the result
// to the bound runtime.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Record, error) {
	changes, err := s.buildChanges(id, input)
	if err != nil {
		return Record{}, err
	}

	record, err := s.repo.Update(ctx, id, changes)
	if err != nil {
		return Record{}, err
	}
	if input.Endpoint == nil && input.Credential == nil {
		return record, nil
	}

	return record, s.apply(ctx, record)
}

// Enable marks a configuration enabled and propagates the change.
func (s *Service) Enable(ctx context.Context, id string) (Record, error) {
	return s.setEnabled(ctx, id, true)
}

// Disable marks a configuration disabled. Executions already admitted keep
// running; new requests are refused by the bound runtime.
func (s *Service) Disable(ctx context.Context, id string) (Record, error) {
	return s.setEnabled(ctx, id, false)
}

// Validate checks the stored endpoint and credential and reports the result to
// the runtime health observer without changing persisted configuration.
func (s *Service) Validate(ctx context.Context, id string) error {
	version := s.registry.healthCheckVersion(id)
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	credential, err := s.credentialOf(ctx, record)
	if err != nil {
		s.registry.reportHealthCheck(id, version, err)
		return err
	}

	if credential == "" {
		err = ErrNoCredential
		s.registry.reportHealthCheck(id, version, err)
		return err
	}

	probe, ok := s.probes[record.Type]
	if !ok || probe == nil {
		err = ErrProbeUnavailable
		s.registry.reportHealthCheck(id, version, err)
		return err
	}

	probeContext, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	err = probe(probeContext, record.Endpoint, credential)
	s.registry.reportHealthCheck(id, version, err)
	return err
}

func (s *Service) setEnabled(ctx context.Context, id string, enabled bool) (Record, error) {
	record, err := s.repo.SetEnabled(ctx, id, enabled)
	if err != nil {
		return Record{}, err
	}

	return record, s.apply(ctx, record)
}

func (s *Service) buildChanges(id string, input UpdateInput) (Changes, error) {
	changes := Changes{}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if err := validateName(name); err != nil {
			return Changes{}, err
		}

		changes.Name = &name
	}

	if input.Endpoint != nil {
		endpoint := strings.TrimSpace(*input.Endpoint)
		if err := s.validateEndpoint(endpoint); err != nil {
			return Changes{}, err
		}

		changes.Endpoint = &endpoint
	}

	if input.Credential != nil {
		sealed, err := s.seal(id, *input.Credential)
		if err != nil {
			return Changes{}, err
		}

		changes.Credential = sealed
	}

	return changes, nil
}

// apply pushes the persisted configuration to the runtime bound to its id.
// Configurations without a binding are administrative only.
func (s *Service) apply(ctx context.Context, record Record) error {
	_, bound := s.registry.Resolve(record.ID)
	if !bound {
		return nil
	}

	credential, err := s.credentialOf(ctx, record)
	if err != nil {
		return err
	}

	if err := s.registry.Reconfigure(record.ID, Settings{
		BaseURL: record.Endpoint,
		APIKey:  credential,
		Enabled: record.Enabled,
	}); err != nil {
		return fmt.Errorf("apply provider %s configuration: %w", record.ID, err)
	}

	return nil
}
