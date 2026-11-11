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
	"aihub/internal/concurrency/pgstore"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func wireE0931PipelineHooks(recorder *pipelineEventRecorder) {
	gateway.IntegrationTestHooks.OnRoutingSelected = func(_, _ string) { recorder.record("routing_selected") }
	gateway.IntegrationTestHooks.OnConcurrencyAcquireAccepted = func(_, _, _ string) {
		recorder.record("concurrency_acquired")
	}
	gateway.IntegrationTestHooks.OnResourceAdmissionAccepted = func(_, _ string) {
		recorder.record("resource_admission_accepted")
	}
	gateway.IntegrationTestHooks.OnExecutionStarted = func(_, _ string) { recorder.record("execution_started") }
	gateway.IntegrationTestHooks.OnProviderCalled = func(_, _ string) { recorder.record("provider_called") }
}

func newE0931PipelineHandler(
	t *testing.T,
	pool *pgxpool.Pool,
	resourceService gateway.ResourceAdmitter,
	concurrencyService *concurrency.Service,
	upstreamURL string,
	inactivity time.Duration,
) http.Handler {
	t.Helper()
	if inactivity <= 0 {
		inactivity = time.Minute
	}
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
		},
		inactivity,
		execution.WithObserver(concurrency.NewReleaseObserver(concurrencyService, nil)),
	)

	_ = pool
	return gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, inactivity, &gateway.ResponsesRouting{
		Catalog:    catalogManager,
		Statuses:   func() []providers.Status { return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}} },
		StaleAfter: 24 * time.Hour,
	}, resourceService, concurrencyService, nil)
}

func countActiveConcurrencyLeases(t *testing.T, pool *pgxpool.Pool, now time.Time) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*)::int FROM concurrency_leases WHERE expires_at > $1`, now.UTC()).Scan(&count)
	if err != nil {
		t.Fatalf("count leases: %v", err)
	}
	return count
}

func secondPoolSameDatabase(t *testing.T, first *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	second, err := pgxpool.NewWithConfig(context.Background(), first.Config())
	if err != nil {
		t.Fatalf("second pool: %v", err)
	}
	t.Cleanup(second.Close)
	return second
}

func e0931ConcurrencyService(t *testing.T, pool *pgxpool.Pool, opts ...concurrency.Option) *concurrency.Service {
	t.Helper()
	return concurrency.NewService(pgstore.New(pool), opts...)
}
