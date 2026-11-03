package gateway_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/catalog"
	"aihub/internal/concurrency"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/resources"
)

func wireConcurrencyPipelineHooks(recorder *pipelineEventRecorder) {
	wirePipelineHooks(recorder)
	gateway.IntegrationTestHooks.OnConcurrencyAcquireStarted = func(_, _, _ string) {
		recorder.record("concurrency_acquire_started")
	}
	gateway.IntegrationTestHooks.OnConcurrencyAcquireAccepted = func(_, _, _ string) {
		recorder.record("concurrency_acquire_accepted")
	}
	gateway.IntegrationTestHooks.OnConcurrencyAcquireRejected = func(_, _, _ string) {
		recorder.record("concurrency_acquire_rejected")
	}
	gateway.IntegrationTestHooks.OnConcurrencyAcquireFailed = func(_, _, _ string) {
		recorder.record("concurrency_acquire_failed")
	}
}

func e2eRoutedHandlerWithConcurrency(
	t *testing.T,
	pool *pgxpool.Pool,
	resourceService *resources.Service,
	concurrencyService *concurrency.Service,
	upstreamURL string,
) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newEnabledClient(t, upstreamURL)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, client)

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	catalogManager := gatewayTestCatalog(t, now, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{gatewayTestDescriptor("pipeline-model")}, nil
		},
	})

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			if gateway.IntegrationTestHooks.OnExecutionStarted != nil {
				gateway.IntegrationTestHooks.OnExecutionStarted(target.ProviderID, target.ModelID)
			}
			resp, err := registry.OpenTargetResponses(ctx, providers.ResponsesTarget{
				ProviderID: target.ProviderID,
				ModelID:    target.ModelID,
			}, payload, contentType, sessionID)
			if err == nil && gateway.IntegrationTestHooks.OnProviderCalled != nil {
				gateway.IntegrationTestHooks.OnProviderCalled(target.ProviderID, target.ModelID)
			}
			return resp, err
		}, time.Minute)

	_ = pool
	return gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}}
		},
		StaleAfter: 24 * time.Hour,
	}, resourceService, concurrencyService, nil)
}
