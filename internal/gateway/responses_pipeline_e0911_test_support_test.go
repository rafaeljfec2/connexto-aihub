package gateway_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/catalog"
	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

type pipelineEventRecorder struct {
	mu     sync.Mutex
	events []string
}

func (r *pipelineEventRecorder) record(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, name)
}

func (r *pipelineEventRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	copy(out, r.events)
	return out
}

func assertSubsequence(t *testing.T, events []string, ordered ...string) {
	t.Helper()
	position := 0
	for _, want := range ordered {
		for position < len(events) && events[position] != want {
			position++
		}
		if position >= len(events) {
			t.Fatalf("events = %v, missing ordered step %q", events, want)
		}
		position++
	}
}

func assertAbsent(t *testing.T, events []string, forbidden ...string) {
	t.Helper()
	for _, name := range forbidden {
		for _, event := range events {
			if event == name {
				t.Fatalf("events = %v, must not contain %q", events, name)
			}
		}
	}
}

func resetIntegrationHooks(t *testing.T) {
	t.Helper()
	gateway.IntegrationTestHooks = struct {
		OnRoutingSelected            func(providerID, modelID string)
		OnResourceAdmissionStarted   func(providerID, modelID string)
		OnResourceAdmissionAccepted  func(providerID, modelID string)
		OnResourceAdmissionRejected  func(providerID, modelID string)
		OnResourceAdmissionFailed    func(providerID, modelID string)
		OnConcurrencyAcquireStarted  func(executionID, providerID, modelID string)
		OnConcurrencyAcquireAccepted func(executionID, providerID, modelID string)
		OnConcurrencyAcquireRejected func(executionID, providerID, modelID string)
		OnConcurrencyAcquireFailed   func(executionID, providerID, modelID string)
		OnExecutionStarted           func(providerID, modelID string)
		OnProviderCalled             func(providerID, modelID string)
	}{}
	t.Cleanup(func() {
		gateway.IntegrationTestHooks = struct {
			OnRoutingSelected            func(providerID, modelID string)
			OnResourceAdmissionStarted   func(providerID, modelID string)
			OnResourceAdmissionAccepted  func(providerID, modelID string)
			OnResourceAdmissionRejected  func(providerID, modelID string)
			OnResourceAdmissionFailed    func(providerID, modelID string)
			OnConcurrencyAcquireStarted  func(executionID, providerID, modelID string)
			OnConcurrencyAcquireAccepted func(executionID, providerID, modelID string)
			OnConcurrencyAcquireRejected func(executionID, providerID, modelID string)
			OnConcurrencyAcquireFailed   func(executionID, providerID, modelID string)
			OnExecutionStarted           func(providerID, modelID string)
			OnProviderCalled             func(providerID, modelID string)
		}{}
	})
}

func wirePipelineHooks(recorder *pipelineEventRecorder) {
	gateway.IntegrationTestHooks.OnRoutingSelected = func(_, _ string) { recorder.record("routing_selected") }
	gateway.IntegrationTestHooks.OnResourceAdmissionStarted = func(_, _ string) { recorder.record("resource_admission_started") }
	gateway.IntegrationTestHooks.OnResourceAdmissionAccepted = func(_, _ string) { recorder.record("resource_admission_accepted") }
	gateway.IntegrationTestHooks.OnResourceAdmissionRejected = func(_, _ string) { recorder.record("resource_admission_rejected") }
	gateway.IntegrationTestHooks.OnResourceAdmissionFailed = func(_, _ string) { recorder.record("resource_admission_failed") }
	gateway.IntegrationTestHooks.OnExecutionStarted = func(_, _ string) { recorder.record("execution_started") }
	gateway.IntegrationTestHooks.OnProviderCalled = func(_, _ string) { recorder.record("provider_called") }
}

func e2eMigratedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

func e2eRoutedHandler(
	t *testing.T,
	admitter gateway.ResourceAdmitter,
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

	return gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}}
		},
		StaleAfter: 24 * time.Hour,
	}, admitter, nil, nil)
}

func postRoutedResponses(t *testing.T, handler http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"input":[{"type":"input_text","text":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)
	return rec
}

func globalCounter(t *testing.T, pool *pgxpool.Pool, period time.Time) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(), `
SELECT c.admitted_count FROM resource_period_counters c
JOIN resource_policies p ON p.id = c.policy_id
WHERE p.scope = 'global' AND c.period_start = $1`, period).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}
