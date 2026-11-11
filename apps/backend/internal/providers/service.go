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
	ID                   string
	Name                 string
	Type                 string
	Endpoint             string
	Credential           string
	ManagementCredential string
	Enabled              bool
}

// UpdateInput changes the optional fields of a configuration. A nil field is
// unchanged; an empty Credential is rejected rather than clearing the stored
// secret silently.
type UpdateInput struct {
	Name                 *string
	Endpoint             *string
	Credential           *string
	ManagementCredential *string
}

// Service owns provider administration: validation, credential protection,
// persistence and propagation of the configuration to the bound runtime.
type Service struct {
	repo             Repository
	cipher           *CredentialCipher
	registry         *Registry
	probes           map[string]ProbeFunc
	allowedHosts     map[string]struct{}
	defaultEndpoints map[string]string
}

// NewService builds the administrative service. allowedHosts is the exact
// host allowlist for provider endpoints (AIHUB_PROVIDER_ALLOWED_HOSTS): the
// credential is only ever sent to these hosts.
func NewService(repo Repository, cipher *CredentialCipher, registry *Registry, probe ProbeFunc, allowedHosts []string) *Service {
	return NewServiceWithProbes(repo, cipher, registry, map[string]ProbeFunc{TypeOpenCodeGo: probe}, allowedHosts)
}

// SetDefaultEndpoint configures the baseline URL for an available provider.
func (s *Service) SetDefaultEndpoint(providerID, endpoint string) {
	if s.defaultEndpoints == nil {
		s.defaultEndpoints = make(map[string]string)
	}
	s.defaultEndpoints[providerID] = strings.TrimSpace(endpoint)
}

// AvailableProviders returns all supported provider templates with UI metadata.
func (s *Service) AvailableProviders() []AvailableProvider {
	cursorEndpoint := DefaultConnextoProxyEndpoint
	if ep, ok := s.defaultEndpoints[CursorID]; ok && ep != "" {
		cursorEndpoint = ep
	}
	openCodeEndpoint := "https://opencode.ai/inference/go"
	if ep, ok := s.defaultEndpoints[OpenCodeGoID]; ok && ep != "" {
		openCodeEndpoint = ep
	}
	openRouterEndpoint := OpenRouterEndpoint
	if ep, ok := s.defaultEndpoints[OpenRouterID]; ok && ep != "" {
		openRouterEndpoint = ep
	}
	antigravityEndpoint := DefaultConnextoProxyEndpoint
	if ep, ok := s.defaultEndpoints[AntigravityID]; ok && ep != "" {
		antigravityEndpoint = ep
	}

	return []AvailableProvider{
		{
			ID:                    CursorID,
			Name:                  "Cursor",
			Description:           "Modelos Cursor (Composer, Grok, etc.) via token; coleta de uso no proxy com Gateway API Key e login web assistido.",
			Logo:                  "cursor",
			CredentialType:        "api_key",
			CredentialLabel:       "API Key / Token do Cursor",
			CredentialPlaceholder: "Informe seu token ou chave do Cursor",
			ManagementCredentialType:        "api_key",
			ManagementCredentialLabel:       "Gateway API Key",
			ManagementCredentialPlaceholder: "Chave do gateway do Connexto Proxy (X-Gateway-API-Key)",
			Capabilities:          []string{"models", "responses", "streaming"},
			ConnectionType:        TypeCursor,
			DefaultEndpoint:       cursorEndpoint,
			Enabled:               true,
		},
		{
			ID:                    AntigravityID,
			Name:                  "Antigravity",
			Description:           "Modelos Antigravity via Connexto Proxy (OAuth local no runtime do proxy)",
			Logo:                  "google",
			CredentialType:        "none",
			CredentialLabel:       "",
			CredentialPlaceholder:   "",
			ManagementCredentialType:        "api_key",
			ManagementCredentialLabel:       "Gateway API Key",
			ManagementCredentialPlaceholder: "Chave do gateway do Connexto Proxy (X-Gateway-API-Key)",
			Capabilities:          []string{"models", "responses", "streaming"},
			ConnectionType:        TypeAntigravity,
			DefaultEndpoint:       antigravityEndpoint,
			Enabled:               true,
		},
		{
			ID:                    OpenRouterID,
			Name:                  "OpenRouter",
			Description:           "Roteamento unificado para centenas de modelos de IA",
			Logo:                  "openrouter",
			CredentialType:        "api_key",
			CredentialLabel:       "API Key do OpenRouter",
			CredentialPlaceholder: "sk-or-v1-...",
			Capabilities:          []string{"models", "responses", "streaming"},
			ConnectionType:        TypeOpenRouter,
			DefaultEndpoint:       openRouterEndpoint,
			Enabled:               true,
		},
		{
			ID:                    OpenCodeGoID,
			Name:                  "OpenCode Go",
			Description:           "Runtime de alto desempenho para modelos de código",
			Logo:                  "opencode",
			CredentialType:        "api_key",
			CredentialLabel:       "Token OpenCode Go",
			CredentialPlaceholder: "opencode-...",
			Capabilities:          []string{"models", "responses", "streaming"},
			ConnectionType:        TypeOpenCodeGo,
			DefaultEndpoint:       openCodeEndpoint,
			Enabled:               true,
		},
	}
}

// AvailableProvider returns the available provider descriptor matching id.
func (s *Service) AvailableProvider(id string) (AvailableProvider, bool) {
	for _, p := range s.AvailableProviders() {
		if p.ID == id {
			return p, true
		}
	}
	return AvailableProvider{}, false
}

// Create stores a new configuration and applies it to the bound runtime adapter.
func (s *Service) Create(ctx context.Context, input CreateInput) (Record, error) {
	if input.ID == "" {
		return Record{}, ErrInvalid
	}
	if avail, ok := s.AvailableProvider(input.ID); ok {
		if input.Name == "" {
			input.Name = avail.Name
		}
		if input.Type == "" {
			input.Type = avail.ConnectionType
		}
		if input.Endpoint == "" {
			input.Endpoint = avail.DefaultEndpoint
		}
	}
	if err := s.validateIdentity(input.ID, input.Name, input.Type); err != nil {
		return Record{}, err
	}
	if input.ManagementCredential != "" && input.Type != TypeOpenRouter && input.Type != TypeConnextoProxy && input.Type != TypeCursor && input.Type != TypeAntigravity {
		return Record{}, ErrInvalid
	}

	if err := s.validateEndpoint(strings.TrimSpace(input.Endpoint)); err != nil {
		return Record{}, err
	}

	sealed, err := s.sealOptional(input.ID, input.Credential)
	if err != nil {
		return Record{}, err
	}
	sealedManagement, err := s.sealManagementOptional(input.ID, input.ManagementCredential)
	if err != nil {
		return Record{}, err
	}

	record, err := s.repo.Create(ctx, NewRecord{
		ID:                   input.ID,
		Name:                 strings.TrimSpace(input.Name),
		Type:                 input.Type,
		Endpoint:             strings.TrimSpace(input.Endpoint),
		Enabled:              input.Enabled,
		Credential:           sealed,
		ManagementCredential: sealedManagement,
	})
	if err != nil {
		return Record{}, err
	}

	_ = s.apply(ctx, record)
	return record, nil
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
	if input.ManagementCredential != nil {
		record, err := s.repo.Get(ctx, id)
		if err != nil {
			return Record{}, err
		}
		if record.Type != TypeOpenRouter && record.Type != TypeConnextoProxy && record.Type != TypeCursor && record.Type != TypeAntigravity {
			return Record{}, ErrInvalid
		}
	}

	changes, err := s.buildChanges(id, input)
	if err != nil {
		return Record{}, err
	}

	record, err := s.repo.Update(ctx, id, changes)
	if err != nil {
		return Record{}, err
	}
	if input.Endpoint == nil && input.Credential == nil && input.ManagementCredential == nil {
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

	probe, ok := s.probes[record.Type]
	if !ok || probe == nil {
		err := ErrProbeUnavailable
		s.registry.reportHealthCheck(id, version, err)
		return err
	}

	probeContext, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	switch record.Type {
	case TypeAntigravity:
		gatewayKey, credErr := s.managementCredentialOf(ctx, record)
		if credErr != nil {
			s.registry.reportHealthCheck(id, version, credErr)
			return credErr
		}
		resolved, ok := EffectiveGatewayKeyForAntigravityValidation(gatewayKey)
		if !ok {
			s.registry.reportHealthCheck(id, version, ErrNoGatewayKey)
			return ErrNoGatewayKey
		}
		probeErr := probe(probeContext, record.Endpoint, resolved)
		s.registry.reportHealthCheck(id, version, probeErr)
		return probeErr
	default:
		credential, credErr := s.credentialOf(ctx, record)
		if credErr != nil {
			s.registry.reportHealthCheck(id, version, credErr)
			return credErr
		}
		if credential == "" {
			s.registry.reportHealthCheck(id, version, ErrNoCredential)
			return ErrNoCredential
		}
		probeErr := probe(probeContext, record.Endpoint, credential)
		s.registry.reportHealthCheck(id, version, probeErr)
		return probeErr
	}
}

// ValidateCredential checks a specific credential against the provider's endpoint.
func (s *Service) ValidateCredential(ctx context.Context, id, credential string) error {
	if strings.TrimSpace(credential) == "" {
		return ErrNoCredential
	}
	endpoint := ""
	providerType := ""
	if record, err := s.repo.Get(ctx, id); err == nil {
		endpoint = record.Endpoint
		providerType = record.Type
	} else if avail, ok := s.AvailableProvider(id); ok {
		endpoint = avail.DefaultEndpoint
		providerType = avail.ConnectionType
	} else {
		return ErrNotFound
	}

	probe, ok := s.probes[providerType]
	if !ok || probe == nil {
		return ErrProbeUnavailable
	}

	probeContext, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	return probe(probeContext, endpoint, credential)
}

// ValidateManagementCredential checks a gateway or management key against the provider endpoint.
func (s *Service) ValidateManagementCredential(ctx context.Context, id, managementCredential string) error {
	if strings.TrimSpace(managementCredential) == "" {
		return ErrNoGatewayKey
	}
	endpoint := ""
	providerType := ""
	if record, err := s.repo.Get(ctx, id); err == nil {
		endpoint = record.Endpoint
		providerType = record.Type
	} else if avail, ok := s.AvailableProvider(id); ok {
		endpoint = avail.DefaultEndpoint
		providerType = avail.ConnectionType
	} else {
		return ErrNotFound
	}

	probe, ok := s.probes[providerType]
	if !ok || probe == nil {
		return ErrProbeUnavailable
	}

	probeContext, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	return probe(probeContext, endpoint, managementCredential)
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
	if input.ManagementCredential != nil {
		sealed, err := s.sealManagement(id, *input.ManagementCredential)
		if err != nil {
			return Changes{}, err
		}
		changes.ManagementCredential = sealed
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

	managementKey, err := s.managementCredentialOf(ctx, record)
	if err != nil {
		return err
	}

	settings := Settings{
		BaseURL:         record.Endpoint,
		APIKey:          credential,
		ManagementKey:   managementKey,
		Enabled:         record.Enabled,
		LogicalProviderID: record.ID,
	}
	switch record.Type {
	case TypeAntigravity:
		settings.InferenceOptional = true
	default:
		settings.InferenceOptional = false
	}
	if err := s.registry.Reconfigure(record.ID, settings); err != nil {
		return fmt.Errorf("apply provider %s configuration: %w", record.ID, err)
	}

	return nil
}
