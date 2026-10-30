package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
)

// unsetAIHUBEnv clears AIHUB_* variables so tests observe real defaults, then
// restores them when the test finishes.
func unsetAIHUBEnv(t *testing.T) {
	t.Helper()

	for _, entry := range os.Environ() {
		name, value, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(name, "AIHUB_") {
			continue
		}

		t.Cleanup(func() { _ = os.Setenv(name, value) })
		_ = os.Unsetenv(name)
	}
}

func writeEnvFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	return path
}

func TestDefaultIsValid(t *testing.T) {
	cfg := config.Default()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config must be valid, got: %v", err)
	}

	if got, want := cfg.Server.Address(), "127.0.0.1:8080"; got != want {
		t.Fatalf("Address() = %q, want %q", got, want)
	}
}

func TestLoadUsesDefaults(t *testing.T) {
	unsetAIHUBEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Server.Port != config.Default().Server.Port {
		t.Fatalf("Port = %d, want default %d", cfg.Server.Port, config.Default().Server.Port)
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvServerHost, "localhost")
	t.Setenv(config.EnvServerPort, "7070")
	t.Setenv(config.EnvServerReadTimeout, "10s")
	t.Setenv(config.EnvServerShutdownTimeout, "3s")
	t.Setenv(config.EnvLogLevel, "warn")
	t.Setenv(config.EnvLogFormat, "text")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if got, want := cfg.Server.Address(), "localhost:7070"; got != want {
		t.Fatalf("Address() = %q, want %q", got, want)
	}

	if got, want := cfg.Server.ReadTimeout, 10*time.Second; got != want {
		t.Fatalf("ReadTimeout = %v, want %v", got, want)
	}

	if got, want := cfg.Server.ShutdownTimeout, 3*time.Second; got != want {
		t.Fatalf("ShutdownTimeout = %v, want %v", got, want)
	}

	if got, want := cfg.Server.IdleTimeout, 60*time.Second; got != want {
		t.Fatalf("IdleTimeout = %v, want the default %v preserved", got, want)
	}

	if cfg.Log.Level != config.LogLevelWarn {
		t.Fatalf("Log.Level = %q, want %q", cfg.Log.Level, config.LogLevelWarn)
	}

	if cfg.Log.Format != config.LogFormatText {
		t.Fatalf("Log.Format = %q, want %q", cfg.Log.Format, config.LogFormatText)
	}
}

func TestLoadReadsValuesFromDotEnv(t *testing.T) {
	unsetAIHUBEnv(t)

	path := writeEnvFile(t, "AIHUB_SERVER_PORT=9090\nAIHUB_LOG_LEVEL=debug\n")

	if err := config.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() returned error: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if got, want := cfg.Server.Port, 9090; got != want {
		t.Fatalf("Port = %d, want %d", got, want)
	}

	if cfg.Log.Level != config.LogLevelDebug {
		t.Fatalf("Log.Level = %q, want %q", cfg.Log.Level, config.LogLevelDebug)
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvServerPort, "not-a-number")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() must reject a non-numeric port, got nil error")
	}
}

func TestLoadReportsEveryInvalidEnvironmentValue(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvServerHost, "")
	t.Setenv(config.EnvServerPort, "0")
	t.Setenv(config.EnvLogLevel, "trace")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() must fail, got nil error")
	}

	for _, want := range []string{"server.host", "server.port", "log.level"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to contain %q", err, want)
		}
	}
}

func TestValidateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantSub string
	}{
		{
			name:    "empty host",
			mutate:  func(cfg *config.Config) { cfg.Server.Host = "" },
			wantSub: "server.host must not be empty",
		},
		{
			name:    "invalid host characters",
			mutate:  func(cfg *config.Config) { cfg.Server.Host = "host with spaces" },
			wantSub: "contains invalid character",
		},
		{
			name:    "port out of range",
			mutate:  func(cfg *config.Config) { cfg.Server.Port = 70000 },
			wantSub: "server.port must be between",
		},
		{
			name:    "zero read header timeout",
			mutate:  func(cfg *config.Config) { cfg.Server.ReadHeaderTimeout = 0 },
			wantSub: "server.read_header_timeout must be greater than zero",
		},
		{
			name: "read header timeout above read timeout",
			mutate: func(cfg *config.Config) {
				cfg.Server.ReadHeaderTimeout = 20 * time.Second
				cfg.Server.ReadTimeout = 10 * time.Second
			},
			wantSub: "must not exceed server.read_timeout",
		},
		{
			name:    "invalid log level",
			mutate:  func(cfg *config.Config) { cfg.Log.Level = "verbose" },
			wantSub: "log.level must be one of",
		},
		{
			name:    "invalid log format",
			mutate:  func(cfg *config.Config) { cfg.Log.Format = "xml" },
			wantSub: "log.format must be one of",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.Default()
			test.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() must fail, got nil error")
			}

			if !strings.Contains(err.Error(), test.wantSub) {
				t.Fatalf("error = %q, want it to contain %q", err, test.wantSub)
			}
		})
	}
}

func TestUnknownEnvVariablesReportsTypos(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv("AIHUB_SERVER_PROT", "8080")
	t.Setenv(config.EnvServerPort, "8080")
	t.Setenv("NOT_AIHUB_PREFIX", "ignored")

	unknown := config.UnknownEnvVariables()

	if len(unknown) != 1 || unknown[0] != "AIHUB_SERVER_PROT" {
		t.Fatalf("UnknownEnvVariables() = %v, want [AIHUB_SERVER_PROT]", unknown)
	}
}

func TestAddressSupportsIPv6(t *testing.T) {
	server := config.Default().Server
	server.Host = "::1"

	if got, want := server.Address(), "[::1]:8080"; got != want {
		t.Fatalf("Address() = %q, want %q", got, want)
	}
}
