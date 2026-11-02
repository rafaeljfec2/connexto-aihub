package config_test

import (
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
)

func TestLoadUsesDatabaseDefaults(t *testing.T) {
	unsetAIHUBEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Database.URL != "" {
		t.Fatalf("database.url = %q, want empty", cfg.Database.URL)
	}
	if cfg.Database.MaxConns != 4 {
		t.Fatalf("database.max_conns = %d, want 4", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 0 {
		t.Fatalf("database.min_conns = %d, want 0", cfg.Database.MinConns)
	}
	if cfg.Database.ConnectTimeout != 5*time.Second {
		t.Fatalf("database.connect_timeout = %v, want 5s", cfg.Database.ConnectTimeout)
	}
	if cfg.Catalog.Persistence != config.EnvCatalogPersistenceFile {
		t.Fatalf("catalog.persistence = %q, want %q without a database URL",
			cfg.Catalog.Persistence, config.EnvCatalogPersistenceFile)
	}
}

func TestLoadDerivesPostgresPersistenceFromDatabaseURL(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvDatabaseURL, "postgres://localhost:5432/ai_hub")
	setTestCredentialKey(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Catalog.Persistence != config.EnvCatalogPersistencePostgres {
		t.Fatalf("catalog.persistence = %q, want %q when a database URL is set",
			cfg.Catalog.Persistence, config.EnvCatalogPersistencePostgres)
	}
}

func TestLoadAppliesDatabaseEnvironmentOverrides(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvDatabaseURL, "postgresql://localhost:5432/ai_hub")
	setTestCredentialKey(t)
	t.Setenv(config.EnvDatabaseMaxConns, "8")
	t.Setenv(config.EnvDatabaseMinConns, "2")
	t.Setenv(config.EnvDatabaseConnectTimeout, "2s")
	t.Setenv(config.EnvCatalogPersistence, "postgres")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Database.MaxConns != 8 || cfg.Database.MinConns != 2 || cfg.Database.ConnectTimeout != 2*time.Second {
		t.Fatalf("database config = %+v, want the environment overrides", cfg.Database)
	}
}

func TestLoadRejectsInvalidDatabaseConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		urlKey  string
		url     string
		key     string
		value   string
		wantSub string
	}{
		{name: "empty persistence without URL", key: config.EnvCatalogPersistence, value: "postgres", wantSub: "AIHUB_DATABASE_URL"},
		{name: "unknown persistence", key: config.EnvCatalogPersistence, value: "sqlite", wantSub: "catalog.persistence"},
		{name: "max conns below range", key: config.EnvDatabaseMaxConns, value: "0", wantSub: "database.max_conns"},
		{name: "max conns above range", key: config.EnvDatabaseMaxConns, value: "65", wantSub: "database.max_conns"},
		{name: "max conns unparsable", key: config.EnvDatabaseMaxConns, value: "many", wantSub: "database.max_conns"},
		{name: "min conns above range", key: config.EnvDatabaseMinConns, value: "33", wantSub: "database.min_conns"},
		{name: "min conns above max conns", key: config.EnvDatabaseMinConns, value: "4", wantSub: "database.min_conns"},
		{name: "connect timeout below range", key: config.EnvDatabaseConnectTimeout, value: "10ms", wantSub: "database.connect_timeout"},
		{name: "connect timeout above range", key: config.EnvDatabaseConnectTimeout, value: "61s", wantSub: "database.connect_timeout"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unsetAIHUBEnv(t)

			if test.urlKey != "" {
				t.Setenv(test.urlKey, test.url)
			}
			t.Setenv(test.key, test.value)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() must reject %s=%s", test.key, test.value)
			}
			if !strings.Contains(err.Error(), test.wantSub) {
				t.Fatalf("error = %q, want it to contain %q", err, test.wantSub)
			}
		})
	}
}

func TestLoadRejectsInvalidDatabaseURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "wrong scheme", url: "mysql://localhost:5432/ai_hub"},
		{name: "empty host", url: "postgres:///ai_hub"},
		{name: "key-value DSN form", url: "host=localhost user=ai_hub dbname=ai_hub"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unsetAIHUBEnv(t)
			t.Setenv(config.EnvDatabaseURL, test.url)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() must reject %s=%q", config.EnvDatabaseURL, test.url)
			}
			if !strings.Contains(err.Error(), "AIHUB_DATABASE_URL") {
				t.Fatalf("error = %q, want it to identify AIHUB_DATABASE_URL", err)
			}
		})
	}
}
