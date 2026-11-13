// Package config defines, loads and validates AI Hub configuration.
//
// Configuration comes from two sources only: built-in defaults and environment
// variables. A dotenv file is loaded into the environment by LoadDotEnv before
// Load runs, so real environment variables still take precedence.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config is the root configuration.
type Config struct {
	Server      Server
	Log         Log
	Provider    Provider
	Catalog     Catalog
	Gateway     Gateway
	Records     Records
	Database    Database
	Credentials Credentials
	UI          UI
	Auth        Auth
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
			ConnextoProxy: ProviderConnextoProxy{
				BaseURL: "http://127.0.0.1:9090/v1",
			},
			AllowedEndpointHosts: append([]string(nil), defaultProviderAllowedHosts...),
		},
		Catalog: Catalog{
			StaleAfter:  24 * time.Hour,
			SyncTimeout: 10 * time.Second,
			Persistence: EnvCatalogPersistenceFile,
		},
		Gateway:  defaultGateway(),
		Records:  defaultRecords(),
		Database: defaultDatabase(),
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

	if err := applyDatabaseEnvironment(&cfg); err != nil {
		return Config{}, err
	}

	if err := applyCredentialEnvironment(&cfg); err != nil {
		return Config{}, err
	}

	if err := applyProviderHostsEnvironment(&cfg); err != nil {
		return Config{}, err
	}

	resolvePersistenceDefaults(&cfg)

	if cfg.Catalog.DataDir == "" {
		dir, err := defaultDataDir()
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", EnvDataDir, err)
		}

		cfg.Catalog.DataDir = dir
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
		if name == EnvHTTPAccessLog {
			continue
		}

		if _, known := knownEnvKeys[name]; !known {
			unknown = append(unknown, name)
		}
	}

	return unknown
}
