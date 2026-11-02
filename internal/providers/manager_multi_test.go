package providers_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"aihub/internal/providers"
)

type modelListTestAdapter struct {
	*routedTestAdapter
	status int
	hits   int
}

func (a *modelListTestAdapter) ListModels(context.Context) (*http.Response, error) {
	a.hits++
	return &http.Response{StatusCode: a.status, Body: io.NopCloser(http.NoBody)}, nil
}

func TestRegistryManagerTracksProviderAvailabilityIndependently(t *testing.T) {
	registry := providers.NewRegistry()
	openCode := &modelListTestAdapter{routedTestAdapter: &routedTestAdapter{}, status: http.StatusOK}
	openRouter := &modelListTestAdapter{routedTestAdapter: &routedTestAdapter{}, status: http.StatusUnauthorized}
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.OpenRouterID, openRouter)
	if err := openCode.Reconfigure(providers.Settings{BaseURL: "https://opencode.example.test", APIKey: "opencode-test-key", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := openRouter.Reconfigure(providers.Settings{BaseURL: "https://openrouter.example.test", APIKey: "openrouter-test-key", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	manager := providers.NewRegistryManager(nil, registry)
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	statuses := manager.Statuses()
	if len(statuses) != 2 || statuses[0].ID != providers.OpenCodeGoID || statuses[0].State != providers.StateHealthy ||
		statuses[1].ID != providers.OpenRouterID || statuses[1].State != providers.StateUnhealthy {
		t.Fatalf("statuses = %+v", statuses)
	}
	if openCode.hits != 1 || openRouter.hits != 1 {
		t.Fatalf("probe hits = %d/%d", openCode.hits, openRouter.hits)
	}
	if err := openRouter.Reconfigure(providers.Settings{BaseURL: "https://openrouter.example.test", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	statuses = manager.Statuses()
	if statuses[1].Enabled || statuses[1].State != providers.StateDisabled || statuses[0].State != providers.StateHealthy {
		t.Fatalf("disable affected wrong provider: %+v", statuses)
	}
}
