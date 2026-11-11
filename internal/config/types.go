package config

import (
	"log/slog"
	"net"
	"strconv"
	"time"
)

// Environment variables read by the application.
const (
	EnvServerHost                = "AIHUB_SERVER_HOST"
	EnvServerPort                = "AIHUB_SERVER_PORT"
	EnvServerReadHeaderTimeout   = "AIHUB_SERVER_READ_HEADER_TIMEOUT"
	EnvServerReadTimeout         = "AIHUB_SERVER_READ_TIMEOUT"
	EnvServerWriteTimeout        = "AIHUB_SERVER_WRITE_TIMEOUT"
	EnvServerIdleTimeout         = "AIHUB_SERVER_IDLE_TIMEOUT"
	EnvServerShutdownTimeout     = "AIHUB_SERVER_SHUTDOWN_TIMEOUT"
	EnvLogLevel                  = "AIHUB_LOG_LEVEL"
	EnvLogFormat                 = "AIHUB_LOG_FORMAT"
	EnvProviderOpenCodeGoBaseURL = "AIHUB_PROVIDER_OPENCODEGO_BASE_URL"
	EnvProviderOpenCodeGoAPIKey  = "AIHUB_PROVIDER_OPENCODEGO_API_KEY"
	EnvProviderConnextoProxyBaseURL = "AIHUB_PROVIDER_CONNEXTO_PROXY_BASE_URL"
	EnvConnextoProxyGatewayKey      = "AIHUB_CONNEXTO_PROXY_GATEWAY_KEY"
	EnvDataDir                   = "AIHUB_DATA_DIR"
	EnvCatalogStaleAfter         = "AIHUB_CATALOG_STALE_AFTER"
	EnvCatalogSyncTimeout        = "AIHUB_CATALOG_SYNC_TIMEOUT"
	EnvGatewayAPIKey             = "AIHUB_GATEWAY_API_KEY"
	EnvAdminAPIKey               = "AIHUB_ADMIN_API_KEY"
	EnvGatewayAllowedHosts       = "AIHUB_GATEWAY_ALLOWED_HOSTS"
	EnvGatewayAllowedOrigins     = "AIHUB_GATEWAY_ALLOWED_ORIGINS"
	EnvGatewayMaxBodyBytes       = "AIHUB_GATEWAY_MAX_BODY_BYTES"
	EnvRecordsMaxAge             = "AIHUB_RECORDS_MAX_AGE"
	EnvRecordsMaxCount           = "AIHUB_RECORDS_MAX_COUNT"
	EnvRecordsMaxStorageBytes    = "AIHUB_RECORDS_MAX_STORAGE_BYTES"
	EnvRecordsSegmentBytes       = "AIHUB_RECORDS_SEGMENT_BYTES"
	EnvRecordsRetentionInterval  = "AIHUB_RECORDS_RETENTION_INTERVAL"
	EnvAutoGate0Enforced         = "AIHUB_AUTO_GATE0_ENFORCED"
)

// knownEnvKeys lists every variable the application consumes.
var knownEnvKeys = map[string]struct{}{
	EnvServerHost:                {},
	EnvServerPort:                {},
	EnvServerReadHeaderTimeout:   {},
	EnvServerReadTimeout:         {},
	EnvServerWriteTimeout:        {},
	EnvServerIdleTimeout:         {},
	EnvServerShutdownTimeout:     {},
	EnvLogLevel:                  {},
	EnvLogFormat:                 {},
	EnvProviderOpenCodeGoBaseURL:    {},
	EnvProviderOpenCodeGoAPIKey:     {},
	EnvProviderConnextoProxyBaseURL: {},
	EnvConnextoProxyGatewayKey:      {},
	EnvDataDir:                      {},
	EnvCatalogStaleAfter:         {},
	EnvCatalogSyncTimeout:        {},
	EnvGatewayAPIKey:             {},
	EnvAdminAPIKey:               {},
	EnvProviderAllowedHosts:      {},
	EnvGatewayAllowedHosts:       {},
	EnvGatewayAllowedOrigins:     {},
	EnvGatewayMaxBodyBytes:       {},
	EnvRecordsMaxAge:             {},
	EnvRecordsMaxCount:           {},
	EnvRecordsMaxStorageBytes:    {},
	EnvRecordsSegmentBytes:       {},
	EnvCredentialKey:             {},
	EnvRecordsRetentionInterval:  {},
	EnvDatabaseURL:               {},
	EnvDatabaseMaxConns:          {},
	EnvDatabaseMinConns:          {},
	EnvDatabaseConnectTimeout:    {},
	EnvCatalogPersistence:        {},
	EnvOperatorPassphrase:        {},
	EnvUISessionSecret:           {},
	EnvUISessionTTL:              {},
	EnvUISecureCookies:           {},
	EnvServeEmbeddedUI:           {},
	EnvAutoGate0Enforced:         {},
}

// LogLevel is a validated log level accepted by the application.
type LogLevel string

// Supported log levels.
const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Valid reports whether the level is supported.
func (l LogLevel) Valid() bool {
	switch l {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		return true
	default:
		return false
	}
}

// SlogLevel maps a validated level to its slog equivalent.
func (l LogLevel) SlogLevel() slog.Level {
	switch l {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LogFormat is a validated log output format.
type LogFormat string

// Supported log formats.
const (
	LogFormatJSON   LogFormat = "json"
	LogFormatText   LogFormat = "text"
	LogFormatPretty LogFormat = "pretty"
)

// Valid reports whether the format is supported.
func (f LogFormat) Valid() bool {
	switch f {
	case LogFormatJSON, LogFormatText, LogFormatPretty:
		return true
	default:
		return false
	}
}

// Server groups the HTTP server settings.
type Server struct {
	Host              string
	Port              int
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// Address returns the host:port pair the server binds to.
func (s Server) Address() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

// Log groups the logging settings.
type Log struct {
	Level  LogLevel
	Format LogFormat
}

// ProviderOpenCodeGo groups the settings of the OpenCode Go provider. The
// credential lives only in the environment; an empty key disables the
// provider without failing startup.
type ProviderOpenCodeGo struct {
	BaseURL string
	APIKey  string
}

// Enabled reports whether the provider has a credential configured.
func (p ProviderOpenCodeGo) Enabled() bool {
	return p.APIKey != ""
}

// ProviderConnextoProxy groups the settings of the Connexto Proxy CLI LLMs provider.
type ProviderConnextoProxy struct {
	BaseURL    string
	GatewayKey string
}

// Provider groups every configured integration.
type Provider struct {
	OpenCodeGo    ProviderOpenCodeGo
	ConnextoProxy ProviderConnextoProxy
	// AllowedEndpointHosts are the only hosts the provider credential may be
	// sent to when an endpoint is configured through the administrative API.
	AllowedEndpointHosts []string
}

// Catalog groups the model catalog settings. An empty DataDir means "resolve
// the XDG default during Load". Persistence is resolved in Load (ADR-020):
// file, or postgres when a database URL is configured.
type Catalog struct {
	DataDir     string
	StaleAfter  time.Duration
	SyncTimeout time.Duration
	Persistence string

	persistenceSet bool // true when AIHUB_CATALOG_PERSISTENCE was explicit
}
