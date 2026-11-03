package gateway_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func gatewayTestDescriptor(id string) providers.ModelDescriptor {
	tools := true
	reasoning := true
	structured := true
	return providers.ModelDescriptor{
		ID:                      id,
		Name:                    id,
		InputModalities:         []string{"text"},
		InputModalitiesPresent:  true,
		OutputModalities:        []string{"text"},
		OutputModalitiesPresent: true,
		ToolsSupported:          &tools,
		ReasoningSupported:      &reasoning,
		StructuredOutput:        &structured,
	}
}

func gatewayTestCatalog(t *testing.T, now time.Time, sources map[string]catalog.SourceFunc) *catalog.Manager {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := catalog.NewStore(t.TempDir(), logger)
	statusFn := func() []providers.Status {
		statuses := make([]providers.Status, 0, len(sources))
		for providerID := range sources {
			statuses = append(statuses, providers.Status{
				ID:      providerID,
				Enabled: true,
				State:   providers.StateHealthy,
			})
		}
		return statuses
	}

	manager := catalog.NewManager(logger, statusFn, sources, store, catalog.Options{
		StaleAfter:  24 * time.Hour,
		SyncTimeout: time.Second,
		Now:         func() time.Time { return now },
	})

	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("catalog Start() error = %v", err)
	}

	for providerID := range sources {
		var err error
		for attempt := 0; attempt < 100; attempt++ {
			_, err = manager.Sync(ctx, providerID)
			if err == nil {
				break
			}
			if errors.Is(err, catalog.ErrSyncInProgress) {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			break
		}
		if err != nil {
			t.Fatalf("catalog Sync(%q) error = %v", providerID, err)
		}
	}

	t.Cleanup(func() { _ = manager.Stop(context.Background()) })
	return manager
}

func newOpenCodeRoutingHandler(t *testing.T, baseURL string, logger *slog.Logger, inactivity time.Duration, modelIDs ...string) http.Handler {
	t.Helper()
	return newOpenCodeRoutingHandlerWithResources(t, baseURL, logger, inactivity, nil, nil, modelIDs...)
}

func newOpenCodeRoutingHandlerWithResources(
	t *testing.T,
	baseURL string,
	logger *slog.Logger,
	inactivity time.Duration,
	resourceAdmission gateway.ResourceAdmitter,
	openerCalls *atomic.Int32,
	modelIDs ...string,
) http.Handler {
	t.Helper()

	client := newEnabledClient(t, baseURL)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, client)

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	descriptors := make([]providers.ModelDescriptor, 0, len(modelIDs))
	for _, id := range modelIDs {
		descriptors = append(descriptors, gatewayTestDescriptor(id))
	}

	catalogManager := gatewayTestCatalog(t, now, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return descriptors, nil
		},
	})

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			if openerCalls != nil {
				openerCalls.Add(1)
			}
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{
				ProviderID: target.ProviderID,
				ModelID:    target.ModelID,
			}, payload, contentType, sessionID)
		}, inactivity)

	return gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, inactivity, &gateway.ResponsesRouting{
		Catalog:    catalogManager,
		Statuses:   func() []providers.Status { return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}} },
		StaleAfter: 24 * time.Hour,
	}, resourceAdmission, nil, nil)
}

func newRoutedResponsesHandler(t *testing.T, openCodeEndpoint, openRouterEndpoint string) (http.Handler, *providers.Client, *providers.OpenRouterClient) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	openCode, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: openCodeEndpoint, APIKey: openCodeTestKey}, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	openRouter, err := providers.NewOpenRouterClient(providers.Settings{BaseURL: openRouterEndpoint + "/api/v1", APIKey: openRouterTestKey, Enabled: true}, nil)
	if err != nil {
		t.Fatalf("NewOpenRouterClient() error = %v", err)
	}
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.OpenRouterID, openRouter)

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	catalogManager := gatewayTestCatalog(t, now, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{gatewayTestDescriptor("opencode-default-model")}, nil
		},
		providers.OpenRouterID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{gatewayTestDescriptor("anthropic/claude-sonnet-4.5")}, nil
		},
	})

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		}, time.Minute)

	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{
				{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy},
				{ID: providers.OpenRouterID, Enabled: true, State: providers.StateHealthy},
			}
		},
		StaleAfter: 24 * time.Hour,
	}, nil, nil, nil)

	return handler, openCode, openRouter
}
