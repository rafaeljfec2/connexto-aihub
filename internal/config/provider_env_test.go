package config_test

import (
	"strings"
	"testing"

	"aihub/internal/config"
)

func TestDefaultProviderOpenCodeGoOptions(t *testing.T) {
	cfg := config.Default()

	provider := cfg.Provider.OpenCodeGo

	if got, want := provider.BaseURL, "https://opencode.ai/inference/go"; got != want {
		t.Fatalf("OpenCodeGo.BaseURL = %q, want %q", got, want)
	}

	if provider.APIKey != "" {
		t.Fatalf("OpenCodeGo.APIKey = %q, want empty", provider.APIKey)
	}

	if provider.Enabled() {
		t.Fatalf("provider must be disabled without a credential")
	}
}

func TestLoadProviderOpenCodeGoEnvironmentOverrides(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvProviderOpenCodeGoBaseURL, "https://upstream.test/go")
	t.Setenv(config.EnvProviderOpenCodeGoAPIKey, "oc_sk_test")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	provider := cfg.Provider.OpenCodeGo
	if got, want := provider.BaseURL, "https://upstream.test/go"; got != want {
		t.Fatalf("OpenCodeGo.BaseURL = %q, want %q", got, want)
	}

	if got, want := provider.APIKey, "oc_sk_test"; got != want {
		t.Fatalf("OpenCodeGo.APIKey = %q, want %q", got, want)
	}

	if !provider.Enabled() {
		t.Fatalf("provider must be enabled with a credential present")
	}
}

func TestLoadRejectsInvalidProviderOpenCodeGoBaseURL(t *testing.T) {
	unsetAIHUBEnv(t)

	cases := map[string]string{
		"scheme missing":         "example.com/inference/go",
		"scheme unsupported":     "ftp://example.com/go",
		"host missing":           "https://",
		"user info embedded":     "https://key@example.com/go",
		"query string forbidden": "https://example.com/go?x=1",
		"fragment forbidden":     "https://example.com/go#frag",
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(config.EnvProviderOpenCodeGoBaseURL, value)
			t.Setenv(config.EnvProviderOpenCodeGoAPIKey, "oc_sk_test")

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() accepted invalid base URL %q", value)
			}

			if !strings.Contains(err.Error(), "provider.opencode-go.base_url") {
				t.Fatalf("error %v does not point at provider.opencode-go.base_url", err)
			}
		})
	}
}

func TestDefaultProviderConnextoProxyOptions(t *testing.T) {
	cfg := config.Default()

	provider := cfg.Provider.ConnextoProxy

	if got, want := provider.BaseURL, "http://127.0.0.1:9090/v1"; got != want {
		t.Fatalf("ConnextoProxy.BaseURL = %q, want %q", got, want)
	}

	if provider.GatewayKey != "" {
		t.Fatalf("ConnextoProxy.GatewayKey = %q, want empty", provider.GatewayKey)
	}
}

func TestLoadProviderConnextoProxyEnvironmentOverrides(t *testing.T) {
	unsetAIHUBEnv(t)

	t.Setenv(config.EnvProviderConnextoProxyBaseURL, "http://127.0.0.1:9095/v1")
	t.Setenv(config.EnvConnextoProxyGatewayKey, "test-gw-key-override")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	provider := cfg.Provider.ConnextoProxy
	if got, want := provider.BaseURL, "http://127.0.0.1:9095/v1"; got != want {
		t.Fatalf("ConnextoProxy.BaseURL = %q, want %q", got, want)
	}

	if got, want := provider.GatewayKey, "test-gw-key-override"; got != want {
		t.Fatalf("ConnextoProxy.GatewayKey = %q, want %q", got, want)
	}
}

func TestLoadRejectsInvalidProviderConnextoProxyBaseURL(t *testing.T) {
	unsetAIHUBEnv(t)

	cases := map[string]string{
		"scheme missing":         "127.0.0.1:9090/v1",
		"scheme unsupported":     "ftp://127.0.0.1:9090/v1",
		"host missing":           "http://",
		"user info embedded":     "http://key@127.0.0.1:9090/v1",
		"query string forbidden": "http://127.0.0.1:9090/v1?x=1",
		"fragment forbidden":     "http://127.0.0.1:9090/v1#frag",
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(config.EnvProviderConnextoProxyBaseURL, value)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() accepted invalid base URL %q", value)
			}

			if !strings.Contains(err.Error(), "provider.connexto-proxy.base_url") {
				t.Fatalf("error %v does not point at provider.connexto-proxy.base_url", err)
			}
		})
	}
}
