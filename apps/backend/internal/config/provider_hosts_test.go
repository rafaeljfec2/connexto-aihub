package config_test

import (
	"reflect"
	"testing"

	"aihub/internal/config"
)

func TestLoadDefaultsProviderEndpointAllowlistToSupportedProviders(t *testing.T) {
	unsetAIHUBEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if want := []string{"opencode.ai", "openrouter.ai", "127.0.0.1", "localhost"}; !reflect.DeepEqual(cfg.Provider.AllowedEndpointHosts, want) {
		t.Fatalf("allowed endpoint hosts = %v, want %v", cfg.Provider.AllowedEndpointHosts, want)
	}
}

func TestLoadParsesProviderEndpointAllowlist(t *testing.T) {
	unsetAIHUBEnv(t)
	t.Setenv(config.EnvProviderAllowedHosts, "opencode.ai, Backup.Example.TEST")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if len(cfg.Provider.AllowedEndpointHosts) != 2 {
		t.Fatalf("allowed endpoint hosts = %v, want two entries", cfg.Provider.AllowedEndpointHosts)
	}
}

func TestLoadRejectsMalformedProviderEndpointAllowlist(t *testing.T) {
	for _, value := range []string{"opencode.ai:443", "https://opencode.ai", "*.example.test", ""} {
		unsetAIHUBEnv(t)
		t.Setenv(config.EnvProviderAllowedHosts, value)

		if _, err := config.Load(); err == nil {
			t.Fatalf("Load() accepted %q", value)
		}
	}
}
