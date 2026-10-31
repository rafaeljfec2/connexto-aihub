package config_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
)

func TestCatalogDefaults(t *testing.T) {
	unsetAIHUBEnv(t)
	t.Setenv("XDG_DATA_HOME", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !filepath.IsAbs(cfg.Catalog.DataDir) {
		t.Fatalf("data dir = %q, want absolute path", cfg.Catalog.DataDir)
	}

	if filepath.Base(cfg.Catalog.DataDir) != "ai-hub" {
		t.Fatalf("data dir = %q, want .../ai-hub", cfg.Catalog.DataDir)
	}

	if cfg.Catalog.StaleAfter != 24*time.Hour {
		t.Fatalf("stale after = %v, want 24h", cfg.Catalog.StaleAfter)
	}

	if cfg.Catalog.SyncTimeout != 10*time.Second {
		t.Fatalf("sync timeout = %v, want 10s", cfg.Catalog.SyncTimeout)
	}
}

func TestCatalogUsesXDGDataHome(t *testing.T) {
	unsetAIHUBEnv(t)

	xdg := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdg)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := filepath.Join(xdg, "ai-hub")
	if cfg.Catalog.DataDir != want {
		t.Fatalf("data dir = %q, want %q", cfg.Catalog.DataDir, want)
	}
}

func TestCatalogEnvOverrides(t *testing.T) {
	unsetAIHUBEnv(t)

	dir := t.TempDir()
	t.Setenv(config.EnvDataDir, dir)
	t.Setenv(config.EnvCatalogStaleAfter, "1h")
	t.Setenv(config.EnvCatalogSyncTimeout, "5s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Catalog.DataDir != dir {
		t.Fatalf("data dir = %q, want %q", cfg.Catalog.DataDir, dir)
	}

	if cfg.Catalog.StaleAfter != time.Hour {
		t.Fatalf("stale after = %v, want 1h", cfg.Catalog.StaleAfter)
	}

	if cfg.Catalog.SyncTimeout != 5*time.Second {
		t.Fatalf("sync timeout = %v, want 5s", cfg.Catalog.SyncTimeout)
	}
}

func TestCatalogRejectsRelativeDataDir(t *testing.T) {
	unsetAIHUBEnv(t)
	t.Setenv(config.EnvDataDir, "relative/path")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want relative path rejection")
	}

	if !strings.Contains(err.Error(), config.EnvDataDir) {
		t.Fatalf("error = %v, want %s", err, config.EnvDataDir)
	}
}

func TestCatalogRejectsInvalidDurations(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "stale after", key: config.EnvCatalogStaleAfter, value: "0s"},
		{name: "sync timeout", key: config.EnvCatalogSyncTimeout, value: "nope"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			unsetAIHUBEnv(t)
			t.Setenv(testCase.key, testCase.value)

			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() error = nil, want invalid %s", testCase.key)
			}
		})
	}
}

func TestCatalogDefaultDataDirRequiresHome(t *testing.T) {
	unsetAIHUBEnv(t)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want home resolution error")
	}
}
