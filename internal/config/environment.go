package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// defaultDataDir resolves the local data directory from the XDG base directory
// specification.
func defaultDataDir() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", errors.New("XDG_DATA_HOME must be an absolute path")
		}

		return filepath.Join(dir, "ai-hub"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(home, ".local", "share", "ai-hub"), nil
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

	if value, ok := os.LookupEnv(EnvDataDir); ok {
		cfg.Catalog.DataDir = value
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
		{EnvCatalogStaleAfter, &cfg.Catalog.StaleAfter},
		{EnvCatalogSyncTimeout, &cfg.Catalog.SyncTimeout},
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

	if err := applyGatewayEnvironment(&cfg.Gateway); err != nil {
		return err
	}

	if err := applyRecordsEnvironment(&cfg.Records); err != nil {
		return err
	}

	return nil
}
