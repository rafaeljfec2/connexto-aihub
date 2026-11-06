package providers

import (
	"errors"
	"regexp"
)

// TypeOpenCodeGo is the adapter type of the OpenCode Go integration.
const TypeOpenCodeGo = "opencode-go"

// Cursor integration identifiers.
const (
	CursorID   = "cursor"
	TypeCursor = "cursor"
)

// Connexto Proxy service identifiers. The proxy is a shared HTTP transport service,
// not a logical provider. Logical providers (cursor today; more later) bind to the
// same ConnextoProxyClient in the registry.
const (
	// LegacyTransportServiceID is the retired registry/catalog id for the proxy
	// service itself. Requests or persisted rows using this id should map to cursor
	// or be removed; do not bind this id in the provider registry.
	LegacyTransportServiceID = "connexto-proxy-cli-llms"

	// ConnextoProxyID is deprecated; use LegacyTransportServiceID.
	ConnextoProxyID = LegacyTransportServiceID

	TypeConnextoProxy            = "connexto-proxy"
	DefaultConnextoProxyEndpoint = "http://127.0.0.1:9090/v1"
)

// NormalizeLogicalProviderID maps retired transport ids to logical provider ids.
func NormalizeLogicalProviderID(providerID string) string {
	if providerID == LegacyTransportServiceID {
		return CursorID
	}

	return providerID
}

// IsLegacyTransportServiceID reports ids that must not appear as registry providers.
func IsLegacyTransportServiceID(providerID string) bool {
	return providerID == LegacyTransportServiceID
}

// AvailableProvider describes a supported provider that can be configured by users.
type AvailableProvider struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Logo                  string   `json:"logo"`
	CredentialType        string   `json:"credential_type"`
	CredentialLabel       string   `json:"credential_label"`
	CredentialPlaceholder string   `json:"credential_placeholder"`
	Capabilities          []string `json:"capabilities"`
	ConnectionType        string   `json:"connection_type"`
	DefaultEndpoint       string   `json:"default_endpoint"`
	Enabled               bool     `json:"enabled"`
}

var (
	ErrInvalid          = errors.New("invalid provider configuration")
	ErrNoCredential     = errors.New("provider has no credential configured")
	ErrProbeUnavailable = errors.New("provider connectivity validation is not supported")
	idPattern           = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
)

const (
	maxNameLength       = 100
	maxCredentialLength = 4096
)
