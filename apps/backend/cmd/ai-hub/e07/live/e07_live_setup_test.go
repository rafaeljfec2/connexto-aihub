package live

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"aihub/internal/config"
	"testing"
)

func TestE07ProviderCredentialUsesOnlyProductionVariable(t *testing.T) {
	if value := e07support.E07ProviderCredential([]string{"OPENCODE_API_KEY=synthetic-legacy-value"}); value != "" {
		t.Fatal("historical credential name was accepted as the provider credential")
	}
	if value := e07support.E07ProviderCredential([]string{config.EnvProviderOpenCodeGoAPIKey + "=synthetic-canonical-value"}); value != "synthetic-canonical-value" {
		t.Fatal("production provider credential variable was not resolved")
	}
	if value := e07support.E07ProviderCredential(nil); value != "" {
		t.Fatal("absent provider credential was reported as present")
	}
}
