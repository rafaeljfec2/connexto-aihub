package live

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"aihub/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestE07ReadOnlyDotEnvCredentialPreflight(t *testing.T) {
	if os.Getenv(e07support.E07CredentialPreflightEnvVar) != "1" {
		t.Skip("opt-in read-only credential preflight is disabled")
	}
	root, err := e07support.E07RepositoryRoot("")
	if err != nil {
		t.Fatal("repository root could not be resolved for credential preflight")
	}
	if err := config.LoadDotEnv(filepath.Join(root, ".env")); err != nil {
		t.Fatal("production dotenv parser could not load the read-only credential file")
	}
	if !e07support.E07EnvironmentHasKey(os.Environ(), config.EnvGatewayAPIKey) ||
		!e07support.E07EnvironmentHasKey(os.Environ(), config.EnvProviderOpenCodeGoAPIKey) {
		t.Fatal("required E07 credentials are unavailable")
	}
	t.Log("E07 credential_presence gateway=PRESENT provider=PRESENT")
}

func TestE07GatewayCredentialUsesOnlyConfiguredVariable(t *testing.T) {
	if got := e07support.E07GatewayCredential(nil); got != "" {
		t.Fatal("missing gateway credential was replaced")
	}
	if got := e07support.E07GatewayCredential([]string{"OPENCODE_API_KEY=synthetic-legacy-key"}); got != "" {
		t.Fatal("legacy provider variable was accepted as the gateway credential")
	}
	configured := config.EnvGatewayAPIKey + "=synthetic-gateway-key"
	if got := e07support.E07GatewayCredential([]string{configured}); got != "synthetic-gateway-key" {
		t.Fatal("configured gateway credential was not selected")
	}
}
