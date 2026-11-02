package codex

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"testing"
)

func TestE07CodexEnvironmentIncludesOnlyGatewayCredential(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin",
		"HOME=/home/validation",
		"LANG=C.UTF-8",
		"AIHUB_GATEWAY_API_KEY=hub-gateway-test-value",
		"AIHUB_PROVIDER_OPENCODEGO_API_KEY=provider-test-value",
		"OPENCODE_API_KEY=other-test-value",
	}
	got := e07support.EnvironmentMap(e07support.E07CodexEnvironment(parent, "/tmp/codex-home", "gateway-test-value"))

	if got["AIHUB_CLIENT_KEY"] != "gateway-test-value" {
		t.Fatal("Codex environment is missing its configured gateway key")
	}
	if got["CODEX_HOME"] != "/tmp/codex-home" || got["HOME"] != "/tmp/codex-home" {
		t.Fatal("Codex environment is missing its isolated home")
	}
	for _, key := range []string{"AIHUB_GATEWAY_API_KEY", "AIHUB_PROVIDER_OPENCODEGO_API_KEY", "OPENCODE_API_KEY"} {
		if _, exists := got[key]; exists {
			t.Fatalf("Codex environment contains a Hub-only variable: %s", key)
		}
	}
	if _, exists := got["LANG"]; !exists {
		t.Fatal("Codex environment omitted a required locale variable")
	}
}
