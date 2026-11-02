package providers_test

import (
	"context"
	"testing"

	"aihub/internal/providers"
)

type reconfigurableTestAdapter struct {
	settings providers.Settings
}

func (a *reconfigurableTestAdapter) Reconfigure(settings providers.Settings) error {
	a.settings = settings
	return nil
}

func (a *reconfigurableTestAdapter) Enabled() bool {
	return a.settings.Enabled && a.settings.APIKey != ""
}

func TestConfigModuleRestoresMultipleProviderCredentialsIndependently(t *testing.T) {
	repo := newMemoryRepository()
	cipher := testCipher(t)
	openCode := &reconfigurableTestAdapter{}
	openRouter := &reconfigurableTestAdapter{}
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.OpenRouterID, openRouter)
	service := providers.NewServiceWithProbes(repo, cipher, registry, nil, []string{"opencode.example.test", "openrouter.example.test"})
	defaults := []providers.BootstrapInput{
		{ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo, Endpoint: "https://opencode.example.test/api/v1", Credential: "opencode-test-credential", Enabled: true},
		{ID: providers.OpenRouterID, Name: "OpenRouter", Type: providers.TypeOpenRouter, Endpoint: "https://openrouter.example.test/api/v1", Credential: "openrouter-test-credential", Enabled: true},
	}
	if err := providers.NewConfigModuleSet(service, defaults, nil).Start(context.Background()); err != nil {
		t.Fatalf("initial Start() error = %v", err)
	}
	if openCode.settings.APIKey != defaults[0].Credential || openRouter.settings.APIKey != defaults[1].Credential {
		t.Fatal("initial configuration was not applied to matching provider IDs")
	}
	sealedOpenCode, err := repo.SealedCredential(context.Background(), providers.OpenCodeGoID)
	if err != nil || string(sealedOpenCode) == defaults[0].Credential {
		t.Fatal("OpenCode credential was not stored as ciphertext")
	}
	sealedOpenRouter, err := repo.SealedCredential(context.Background(), providers.OpenRouterID)
	if err != nil || string(sealedOpenRouter) == defaults[1].Credential {
		t.Fatal("OpenRouter credential was not stored as ciphertext")
	}

	restartedOpenCode := &reconfigurableTestAdapter{}
	restartedOpenRouter := &reconfigurableTestAdapter{}
	restartedRegistry := providers.NewRegistry()
	restartedRegistry.Bind(providers.OpenCodeGoID, restartedOpenCode)
	restartedRegistry.Bind(providers.OpenRouterID, restartedOpenRouter)
	restartedService := providers.NewServiceWithProbes(repo, cipher, restartedRegistry, nil, []string{"opencode.example.test", "openrouter.example.test"})
	for index := range defaults {
		defaults[index].Credential = ""
	}
	if err := providers.NewConfigModuleSet(restartedService, defaults, nil).Start(context.Background()); err != nil {
		t.Fatalf("restart Start() error = %v", err)
	}
	if restartedOpenCode.settings.APIKey != "opencode-test-credential" || restartedOpenRouter.settings.APIKey != "openrouter-test-credential" {
		t.Fatal("restart did not recover independent persisted credentials")
	}
	manager := providers.NewRegistryManager(nil, restartedRegistry)
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("provider Manager startup after recovery = %v", err)
	}
	statuses := manager.Statuses()
	if len(statuses) != 2 || statuses[0].State != providers.StateUnknown || statuses[1].State != providers.StateUnknown {
		t.Fatalf("provider states after credential recovery = %+v, want unknown without probes", statuses)
	}
}
