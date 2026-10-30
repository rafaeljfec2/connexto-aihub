// Package config defines, loads and validates AI Hub configuration.
//
// Configuration comes from two sources only: built-in defaults and environment
// variables. A dotenv file is loaded into the environment by LoadDotEnv before
// Load runs, so real environment variables still take precedence.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	EnvProviderOpenCodeGoBaseURL: {},
	EnvProviderOpenCodeGoAPIKey:  {},
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
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

// Valid reports whether the format is supported.
func (f LogFormat) Valid() bool {
	switch f {
	case LogFormatJSON, LogFormatText:
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

// Provider groups every configured integration.
type Provider struct {
	OpenCodeGo ProviderOpenCodeGo
}

// Config is the root configuration.
type Config struct {
	Server   Server
	Log      Log
	Provider Provider
}

// Default returns the baseline configuration used as the starting point for
// environment overrides.
func Default() Config {
	return Config{
		Server: Server{
			Host:              "127.0.0.1",
			Port:              8080,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   10 * time.Second,
		},
		Log: Log{
			Level:  LogLevelInfo,
			Format: LogFormatJSON,
		},
		Provider: Provider{
			OpenCodeGo: ProviderOpenCodeGo{
				BaseURL: "https://opencode.ai/inference/go",
			},
		},
	}
}

// Load builds the configuration from defaults and environment variables and
// validates it. Environment variables are expected to be populated already,
// either by the process or by LoadDotEnv.
func Load() (Config, error) {
	cfg := Default()

	if err := applyEnvironment(&cfg); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// UnknownEnvVariables returns AIHUB_* variables present in the environment that
// the application does not read. They are usually typos, which the previous
// strict schema decoder would have rejected.
func UnknownEnvVariables() []string {
	unknown := make([]string, 0)

	for _, entry := range os.Environ() {
		name, _, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(name, "AIHUB_") {
			continue
		}

		if _, known := knownEnvKeys[name]; !known {
			unknown = append(unknown, name)
		}
	}

	return unknown
}

func applyEnvironment(cfg *Config) error {
	if value, ok := os.LookupEnv(EnvServerHost); ok {
		cfg.Server.Host = value
	}

	if value, ok := os.LookupEnv(EnvServerPort); ok {
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s must be an integer: %w", EnvServerPort, err)
		}
		cfg.Server.Port = port
	}

	if value, ok := os.LookupEnv(EnvLogLevel); ok {
		cfg.Log.Level = LogLevel(value)
	}

	if value, ok := os.LookupEnv(EnvLogFormat); ok {
		cfg.Log.Format = LogFormat(value)
	}

	if value, ok := os.LookupEnv(EnvProviderOpenCodeGoBaseURL); ok {
		cfg.Provider.OpenCodeGo.BaseURL = value
	}

	if value, ok := os.LookupEnv(EnvProviderOpenCodeGoAPIKey); ok {
		cfg.Provider.OpenCodeGo.APIKey = value
	}

	durations := []struct {
		name   string
		target *time.Duration
	}{
		{EnvServerReadHeaderTimeout, &cfg.Server.ReadHeaderTimeout},
		{EnvServerReadTimeout, &cfg.Server.ReadTimeout},
		{EnvServerWriteTimeout, &cfg.Server.WriteTimeout},
		{EnvServerIdleTimeout, &cfg.Server.IdleTimeout},
		{EnvServerShutdownTimeout, &cfg.Server.ShutdownTimeout},
	}

	for _, item := range durations {
		value, ok := os.LookupEnv(item.name)
		if !ok {
			continue
		}

		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}

		*item.target = parsed
	}

	return nil
}

// Validate reports every invalid field in a single error.
func (c Config) Validate() error {
	var errs []error

	if err := validateHost(c.Server.Host); err != nil {
		errs = append(errs, err)
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port))
	}

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{"server.read_header_timeout", c.Server.ReadHeaderTimeout},
		{"server.read_timeout", c.Server.ReadTimeout},
		{"server.write_timeout", c.Server.WriteTimeout},
		{"server.idle_timeout", c.Server.IdleTimeout},
		{"server.shutdown_timeout", c.Server.ShutdownTimeout},
	}

	for _, timeout := range timeouts {
		if timeout.value <= 0 {
			errs = append(errs, fmt.Errorf("%s must be greater than zero", timeout.name))
		}
	}

	if c.Server.ReadHeaderTimeout > c.Server.ReadTimeout {
		errs = append(errs, errors.New("server.read_header_timeout must not exceed server.read_timeout"))
	}

	if !c.Log.Level.Valid() {
		errs = append(errs, fmt.Errorf("log.level must be one of debug, info, warn, error; got %q", c.Log.Level))
	}

	if !c.Log.Format.Valid() {
		errs = append(errs, fmt.Errorf("log.format must be one of json, text; got %q", c.Log.Format))
	}

	if err := ValidateBaseURL(c.Provider.OpenCodeGo.BaseURL); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// ValidateBaseURL rejects provider base URLs that cannot be joined safely with
// the fixed endpoint paths, including URLs with embedded credentials.
func ValidateBaseURL(baseURL string) error {
	const field = "provider.opencode-go.base_url"

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%s scheme must be http or https, got %q", field, parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%s host must not be empty", field)
	}

	if parsed.User != nil {
		return errors.New(field + " must not embed credentials; use the dedicated environment variable")
	}

	if parsed.RawQuery != "" {
		return errors.New(field + " must not carry a query string")
	}

	if parsed.Fragment != "" {
		return errors.New(field + " must not carry a fragment")
	}

	return nil
}

func validateHost(host string) error {
	if host == "" {
		return errors.New("server.host must not be empty")
	}

	if strings.TrimSpace(host) != host {
		return fmt.Errorf("server.host must not contain surrounding spaces: %q", host)
	}

	if net.ParseIP(host) != nil || host == "localhost" {
		return nil
	}

	if len(host) > 253 {
		return fmt.Errorf("server.host is too long: %d characters", len(host))
	}

	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 {
			return fmt.Errorf("server.host %q is not a valid hostname", host)
		}

		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("server.host %q has a label starting or ending with '-'", host)
		}

		for _, char := range label {
			isAllowed := char >= 'a' && char <= 'z' ||
				char >= 'A' && char <= 'Z' ||
				char >= '0' && char <= '9' ||
				char == '-'
			if !isAllowed {
				return fmt.Errorf("server.host %q contains invalid character %q", host, char)
			}
		}
	}

	return nil
}
