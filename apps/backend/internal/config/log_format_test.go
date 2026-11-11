package config_test

import (
	"testing"

	"aihub/internal/config"
)

func TestLoadAcceptsPrettyLogFormat(t *testing.T) {
	unsetAIHUBEnv(t)
	t.Setenv(config.EnvLogFormat, "pretty")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Log.Format != config.LogFormatPretty {
		t.Fatalf("Log.Format = %q, want pretty", cfg.Log.Format)
	}
}
