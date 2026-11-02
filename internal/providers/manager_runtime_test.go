package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/providers"
)

func TestManagerInvalidatesDisabledStateAfterCredentialUpdate(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/v1/models" {
			t.Errorf("probe request = %s %s", request.Method, request.URL.Path)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	endpoint := server.URL + "/api/v1"
	repo := newMemoryRepository()
	registry := providers.NewRegistry()
	adapter, err := providers.NewOpenRouterClient(providers.Settings{
		BaseURL: endpoint,
		Enabled: true,
	}, nil)
	if err != nil {
		t.Fatalf("NewOpenRouterClient: %v", err)
	}
	registry.Bind(providers.OpenRouterID, adapter)
	service := providers.NewServiceWithProbes(repo, testCipher(t), registry, map[string]providers.ProbeFunc{
		providers.TypeOpenRouter: func(context.Context, string, string) error { return nil },
	}, []string{"127.0.0.1"})
	if _, err := service.Bootstrap(ctx, providers.BootstrapInput{
		ID: providers.OpenRouterID, Name: "OpenRouter", Type: providers.TypeOpenRouter,
		Endpoint: endpoint, Enabled: true,
	}); err != nil {
		t.Fatalf("Bootstrap without credential: %v", err)
	}

	manager := providers.NewRegistryManager(nil, registry)
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if got := manager.Statuses()[0]; got.Enabled || got.State != providers.StateDisabled {
		t.Fatalf("initial status = %+v, want disabled", got)
	}

	credential := "sk-or-test-runtime-credential"
	if _, err := service.Update(ctx, providers.OpenRouterID, providers.UpdateInput{Credential: &credential}); err != nil {
		t.Fatalf("Update credential: %v", err)
	}

	got := manager.Statuses()[0]
	if !got.Enabled || got.State != providers.StateUnknown {
		t.Fatalf("status after credential update = %+v, want enabled/unknown", got)
	}
	if err := service.Validate(ctx, providers.OpenRouterID); err != nil {
		t.Fatalf("Validate after credential update: %v", err)
	}
	if got := manager.Statuses()[0]; !got.Enabled || got.State != providers.StateHealthy {
		t.Fatalf("status after successful probe = %+v, want enabled/healthy", got)
	}

	name := "OpenRouter Runtime"
	if _, err := service.Update(ctx, providers.OpenRouterID, providers.UpdateInput{Name: &name}); err != nil {
		t.Fatalf("Update name: %v", err)
	}
	if got := manager.Statuses()[0]; !got.Enabled || got.State != providers.StateHealthy {
		t.Fatalf("status after name update = %+v, want enabled/healthy", got)
	}

	rotatedCredential := "sk-or-test-rotated-credential"
	if _, err := service.Update(ctx, providers.OpenRouterID, providers.UpdateInput{Credential: &rotatedCredential}); err != nil {
		t.Fatalf("Update rotated credential: %v", err)
	}
	if got := manager.Statuses()[0]; !got.Enabled || got.State != providers.StateUnknown {
		t.Fatalf("status after credential rotation = %+v, want enabled/unknown", got)
	}
}
