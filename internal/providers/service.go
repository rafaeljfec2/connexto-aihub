package providers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"aihub/internal/config"
)

// TypeOpenCodeGo is the adapter type of the OpenCode Go integration.
const TypeOpenCodeGo = "opencode-go"

// Validation and credential errors. Their messages never include secrets.
var (
	ErrInvalid      = errors.New("invalid provider configuration")
	ErrNoCredential = errors.New("provider has no credential configured")
)

const (
	maxNameLength       = 100
	maxCredentialLength = 4096
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

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

// BootstrapInput seeds a configuration from the startup environment.
type BootstrapInput struct {
	ID         string
	Name       string
	Type       string
	Endpoint   string
	Credential string
	Enabled    bool
}

// Service owns provider administration: validation, credential protection,
// persistence and propagation of the configuration to the bound runtime.
type Service struct {
	repo         Repository
	cipher       *CredentialCipher
	registry     *Registry
	probe        ProbeFunc
	allowedHosts map[string]struct{}
}

// NewService builds the administrative service. allowedHosts is the exact
// host allowlist for provider endpoints (AIHUB_PROVIDER_ALLOWED_HOSTS): the
// credential is only ever sent to these hosts.
func NewService(repo Repository, cipher *CredentialCipher, registry *Registry, probe ProbeFunc, allowedHosts []string) *Service {
	hosts := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		hosts[strings.ToLower(host)] = struct{}{}
	}

	return &Service{repo: repo, cipher: cipher, registry: registry, probe: probe, allowedHosts: hosts}
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

// Validate checks upstream connectivity with the stored endpoint and
// credential. It changes no state.
func (s *Service) Validate(ctx context.Context, id string) error {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	credential, err := s.credentialOf(ctx, record)
	if err != nil {
		return err
	}

	if credential == "" {
		return ErrNoCredential
	}

	return s.probe(ctx, record.Endpoint, credential)
}

// Bootstrap initializes a configuration from the environment when the store
// has none yet. Once persisted, the stored values are authoritative. The
// resulting configuration is applied to the bound runtime.
func (s *Service) Bootstrap(ctx context.Context, input BootstrapInput) (Record, error) {
	if err := s.validateIdentity(input.ID, input.Name, input.Type); err != nil {
		return Record{}, err
	}

	// Environment settings only matter when they seed a new configuration. A
	// persisted configuration is authoritative, so an unused environment value
	// must not block startup; the persisted endpoint is always re-checked.
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
		ID:         input.ID,
		Name:       strings.TrimSpace(input.Name),
		Type:       input.Type,
		Endpoint:   strings.TrimSpace(input.Endpoint),
		Enabled:    input.Enabled,
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

// Apply propagates the persisted configuration of one id to its bound runtime.
// It is used at startup after the record is loaded from storage.
func (s *Service) Apply(ctx context.Context, id string) error {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	return s.apply(ctx, record)
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
	adapter, bound := s.registry.Resolve(record.ID)
	if !bound {
		return nil
	}

	credential, err := s.credentialOf(ctx, record)
	if err != nil {
		return err
	}

	if err := adapter.Reconfigure(Settings{
		BaseURL: record.Endpoint,
		APIKey:  credential,
		Enabled: record.Enabled,
	}); err != nil {
		return fmt.Errorf("apply provider %s configuration: %w", record.ID, err)
	}

	return nil
}

func (s *Service) credentialOf(ctx context.Context, record Record) (string, error) {
	sealed, err := s.repo.SealedCredential(ctx, record.ID)
	if err != nil {
		return "", err
	}

	if sealed == nil {
		return "", nil
	}

	return s.cipher.Open(record.ID, sealed)
}

func (s *Service) sealOptional(id, credential string) ([]byte, error) {
	if credential == "" {
		return nil, nil
	}

	return s.seal(id, credential)
}

func (s *Service) seal(id, credential string) ([]byte, error) {
	if err := validateCredential(credential); err != nil {
		return nil, err
	}

	return s.cipher.Seal(id, strings.TrimSpace(credential))
}

func (s *Service) validateIdentity(id, name, providerType string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("%w: id must be lowercase letters, digits and hyphens", ErrInvalid)
	}

	if err := validateName(strings.TrimSpace(name)); err != nil {
		return err
	}

	if providerType != TypeOpenCodeGo {
		return fmt.Errorf("%w: unsupported provider type", ErrInvalid)
	}

	return nil
}

func validateName(name string) error {
	if name == "" || len(name) > maxNameLength {
		return fmt.Errorf("%w: name must be 1 to %d characters", ErrInvalid, maxNameLength)
	}

	return nil
}

// validateEndpoint applies the startup base URL rules and additionally
// requires HTTPS for administrative configurations: the bearer credential
// must not travel in cleartext to a remote host. Plain HTTP remains allowed
// for loopback targets (local development and tests).
func (s *Service) validateEndpoint(endpoint string) error {
	if err := config.ValidateBaseURL(endpoint); err != nil {
		return fmt.Errorf("%w: endpoint is not a valid base URL", ErrInvalid)
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("%w: endpoint is not a valid base URL", ErrInvalid)
	}

	if strings.EqualFold(parsed.Scheme, "http") && !isLoopbackHost(parsed.Hostname()) {
		return fmt.Errorf("%w: endpoint must use https unless it targets a loopback host", ErrInvalid)
	}

	if _, allowed := s.allowedHosts[strings.ToLower(parsed.Hostname())]; !allowed {
		return fmt.Errorf("%w: endpoint host is not in AIHUB_PROVIDER_ALLOWED_HOSTS", ErrInvalid)
	}

	return nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

func validateCredential(credential string) error {
	trimmed := strings.TrimSpace(credential)
	if trimmed == "" || len(trimmed) > maxCredentialLength {
		return fmt.Errorf("%w: credential must be 1 to %d characters", ErrInvalid, maxCredentialLength)
	}

	return nil
}
