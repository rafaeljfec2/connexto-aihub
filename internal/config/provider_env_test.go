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
