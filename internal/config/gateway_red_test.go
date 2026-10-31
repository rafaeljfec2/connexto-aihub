package config_test

import (
	"os"
	"strings"
	"testing"

	"aihub/internal/config"
)

const (
	gatewayKeyEnv = "AIHUB_GATEWAY_API_KEY"
	gatewayKey    = "test-gateway-key-0123456789abcdef"
)

func clearGatewayEnv(t *testing.T) {
	t.Helper()

	unsetAIHUBEnv(t)
	_ = os.Unsetenv(gatewayKeyEnv)
}

func TestLoadRequiresGatewayAPIKey(t *testing.T) {
	clearGatewayEnv(t)

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() must reject a missing gateway API key")
	}

	if !strings.Contains(err.Error(), "gateway.api_key") {
		t.Fatalf("error = %q, want it to identify gateway.api_key", err)
	}
}

func TestLoadRejectsInvalidGatewayAPIKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "too short", key: strings.Repeat("a", 31)},
		{name: "too long", key: strings.Repeat("a", 513)},
		{name: "space", key: "test gateway key 0123456789abcdef"},
		{name: "comma", key: "test,gateway,key,0123456789abcdef"},
		{name: "colon", key: "test:gateway:key:0123456789abcdef"},
		{name: "semicolon", key: "test;gateway;key;0123456789abcdef"},
		{name: "unicode", key: "test-gateway-key-0123456789abcdé"},
		{name: "only padding", key: strings.Repeat("=", 32)},
		{name: "internal padding", key: "test-gateway-key-0123456789abc=def"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearGatewayEnv(t)
			t.Setenv(gatewayKeyEnv, test.key)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() must reject the %s key", test.name)
			}

			if !strings.Contains(err.Error(), "gateway.api_key") {
				t.Fatalf("error = %q, want it to identify gateway.api_key", err)
			}
		})
	}
}

func TestLoadRejectsProviderKeyReusedAsGatewayKey(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv(config.EnvProviderOpenCodeGoAPIKey, gatewayKey)
	t.Setenv(gatewayKeyEnv, gatewayKey)

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() must reject a gateway key equal to the provider key")
	}

	if !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("error = %q, want it to explain that the keys must differ", err)
	}

	if strings.Contains(err.Error(), gatewayKey) {
		t.Fatal("the configuration error must not contain the key value")
	}
}

func TestLoadRejectsExplicitlyEmptyAllowedHosts(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv(gatewayKeyEnv, gatewayKey)
	t.Setenv("AIHUB_GATEWAY_ALLOWED_HOSTS", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() must reject an explicitly empty allowed-host list")
	}

	if !strings.Contains(err.Error(), "allowed_hosts") {
		t.Fatalf("error = %q, want it to identify the allowed-host setting", err)
	}
}

func TestLoadRejectsNonLoopbackHostWithoutExplicitAllowedHosts(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv(gatewayKeyEnv, gatewayKey)
	t.Setenv(config.EnvServerHost, "0.0.0.0")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() must reject a non-loopback bind without explicit allowed hosts")
	}

	if !strings.Contains(err.Error(), "allowed_hosts") {
		t.Fatalf("error = %q, want it to identify the allowed-host setting", err)
	}
}

func TestLoadAcceptsNonLoopbackHostWithExplicitAllowedHosts(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv(gatewayKeyEnv, gatewayKey)
	t.Setenv(config.EnvServerHost, "0.0.0.0")
	t.Setenv("AIHUB_GATEWAY_ALLOWED_HOSTS", "localhost,127.0.0.1")

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
}

func TestLoadRejectsInvalidAllowedOrigins(t *testing.T) {
	tests := []string{
		"null",
		"*",
		"https://example.com/path",
		"https://user@example.com",
		"https://example.com?query=1",
		"https://example.com#fragment",
		"ftp://example.com",
	}

	for _, origin := range tests {
		t.Run(origin, func(t *testing.T) {
			clearGatewayEnv(t)
			t.Setenv(gatewayKeyEnv, gatewayKey)
			t.Setenv("AIHUB_GATEWAY_ALLOWED_ORIGINS", origin)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() must reject origin %q", origin)
			}

			if !strings.Contains(err.Error(), "allowed_origins") {
				t.Fatalf("error = %q, want it to identify the allowed-origin setting", err)
			}
		})
	}
}

func TestLoadAcceptsEmptyAllowedOrigins(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv(gatewayKeyEnv, gatewayKey)
	t.Setenv("AIHUB_GATEWAY_ALLOWED_ORIGINS", "")

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
}

func TestLoadRejectsInvalidGatewayBodyLimit(t *testing.T) {
	tests := []string{"1023", "134217729", "not-a-number"}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			clearGatewayEnv(t)
			t.Setenv(gatewayKeyEnv, gatewayKey)
			t.Setenv("AIHUB_GATEWAY_MAX_BODY_BYTES", value)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() must reject body limit %q", value)
			}

			if !strings.Contains(err.Error(), "max_body_bytes") {
				t.Fatalf("error = %q, want it to identify the body-limit setting", err)
			}
		})
	}
}

func TestLoadAggregatesGatewaySemanticErrors(t *testing.T) {
	clearGatewayEnv(t)
	t.Setenv(gatewayKeyEnv, "short")
	t.Setenv(config.EnvServerHost, "0.0.0.0")
	t.Setenv("AIHUB_GATEWAY_MAX_BODY_BYTES", "128")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() must fail")
	}

	for _, want := range []string{"gateway.api_key", "allowed_hosts", "max_body_bytes"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to contain %q", err, want)
		}
	}
}

func TestGatewayConfigurationErrorDoesNotLeakTheKey(t *testing.T) {
	const marker = "leaky-gateway-key-marker"

	clearGatewayEnv(t)
	t.Setenv(gatewayKeyEnv, marker+",not-valid")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() must fail")
	}

	if strings.Contains(err.Error(), marker) {
		t.Fatal("the configuration error must not contain the key value")
	}
}
