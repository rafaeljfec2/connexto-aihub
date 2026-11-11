package pgstore_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/config"
	"aihub/internal/providers"
	"aihub/internal/providers/pgstore"
)

func newMultiProviderService(t *testing.T, pool *pgxpool.Pool, adapters map[string]*captureAdapter) *providers.Service {
	t.Helper()

	cipher, err := providers.NewCredentialCipher(bytes.Repeat([]byte{0x33}, config.CredentialKeyBytes))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}
	registry := providers.NewRegistry()
	for id, adapter := range adapters {
		registry.Bind(id, adapter)
	}
	return providers.NewService(pgstore.New(pool), cipher, registry,
		func(context.Context, string, string) error { return nil },
		[]string{"provider.example.test", "other.example.test"})
}

func TestMultipleProviderConfigurationsRecoverIndependentlyAfterRestartIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()

	openCode := &captureAdapter{}
	openRouter := &captureAdapter{}
	first := newMultiProviderService(t, pool, map[string]*captureAdapter{
		providers.OpenCodeGoID: openCode,
		providers.OpenRouterID: openRouter,
	})
	defaults := []providers.BootstrapInput{
		{
			ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
			Endpoint: testEndpoint, Credential: "opencode-integration-key", Enabled: true,
		},
		{
			ID: providers.OpenRouterID, Name: "OpenRouter", Type: providers.TypeOpenRouter,
			Endpoint: "https://other.example.test/api/v1", Credential: "openrouter-integration-key", Enabled: true,
		},
	}
	for _, input := range defaults {
		if _, err := first.Bootstrap(ctx, input); err != nil {
			t.Fatalf("first Bootstrap(%s): %v", input.ID, err)
		}
	}
	if openCode.last.APIKey != defaults[0].Credential || openRouter.last.APIKey != defaults[1].Credential {
		t.Fatal("initial configurations were not isolated by provider ID")
	}

	openCodeEndpoint := "https://provider.example.test/reconfigured/opencode"
	openRouterEndpoint := "https://other.example.test/reconfigured/openrouter"
	openCodeCredential := "opencode-admin-updated-key"
	openRouterCredential := "openrouter-admin-updated-key"
	if _, err := first.Update(ctx, providers.OpenCodeGoID, providers.UpdateInput{
		Endpoint: &openCodeEndpoint, Credential: &openCodeCredential,
	}); err != nil {
		t.Fatalf("Update OpenCode Go: %v", err)
	}
	if _, err := first.Update(ctx, providers.OpenRouterID, providers.UpdateInput{
		Endpoint: &openRouterEndpoint, Credential: &openRouterCredential,
	}); err != nil {
		t.Fatalf("Update OpenRouter: %v", err)
	}

	poolConfig := pool.Config()
	pool.Close()
	restartedPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("reopen database pool: %v", err)
	}
	t.Cleanup(restartedPool.Close)
	// Fresh adapters, service, and pool model a process restart; the persisted database state wins over bootstrap defaults.
	restartedOpenCode := &captureAdapter{}
	restartedOpenRouter := &captureAdapter{}
	restarted := newMultiProviderService(t, restartedPool, map[string]*captureAdapter{
		providers.OpenCodeGoID: restartedOpenCode,
		providers.OpenRouterID: restartedOpenRouter,
	})
	for _, input := range defaults {
		input.Endpoint = testEndpoint
		input.Credential = "bootstrap-default-after-restart"
		if _, err := restarted.Bootstrap(ctx, input); err != nil {
			t.Fatalf("restart Bootstrap(%s): %v", input.ID, err)
		}
	}

	if restartedOpenCode.last.APIKey != openCodeCredential || restartedOpenCode.last.BaseURL != openCodeEndpoint {
		t.Fatal("restart did not recover OpenCode Go configuration independently")
	}
	if restartedOpenRouter.last.APIKey != openRouterCredential || restartedOpenRouter.last.BaseURL != openRouterEndpoint {
		t.Fatal("restart did not recover OpenRouter configuration independently")
	}
	if !restartedOpenCode.last.Enabled || !restartedOpenRouter.last.Enabled {
		t.Fatal("restart did not recover both providers' enabled state")
	}
}
