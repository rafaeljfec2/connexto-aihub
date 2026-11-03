package gateway_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/catalog"
	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/resources"
	"aihub/internal/resources/pgstore"
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
		Catalog:    catalogManager,
		Statuses:   func() []providers.Status { return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}} },
		StaleAfter: 24 * time.Hour,
	}, admitter, nil)
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

func TestE0911ScenarioAAuthorizedAdmissionPipelineIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	wirePipelineHooks(recorder)

	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	ctx := context.Background()
	if _, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 5, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	handler := e2eRoutedHandler(t, service, upstream.URL)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	events := recorder.snapshot()
	assertSubsequence(t, events,
		"routing_selected",
		"resource_admission_started",
		"resource_admission_accepted",
		"execution_started",
		"provider_called",
	)
	period, _ := resources.UsagePeriodBounds(now)
	if count := globalCounter(t, pool, period); count != 1 {
		t.Fatalf("global counter = %d, want 1", count)
	}
}

func TestE0911ScenarioBQuotaExceededPipelineIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	wirePipelineHooks(recorder)

	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	ctx := context.Background()
	if _, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	var providerCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := providerCalls.Add(1)
		if n > 1 {
			t.Error("provider must not be called when quota rejects")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	handler := e2eRoutedHandler(t, service, upstream.URL)
	first := postRoutedResponses(t, handler)
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d", first.Code)
	}

	recorder.events = nil
	second := postRoutedResponses(t, handler)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d body = %s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Body.String(), "resource_limit_exceeded") {
		t.Fatalf("body = %s", second.Body.String())
	}

	events := recorder.snapshot()
	assertSubsequence(t, events, "routing_selected", "resource_admission_started", "resource_admission_rejected")
	assertAbsent(t, events, "execution_started", "provider_called", "resource_admission_accepted")

	period, _ := resources.UsagePeriodBounds(now)
	if count := globalCounter(t, pool, period); count != 1 {
		t.Fatalf("global counter = %d, want 1 after rejection", count)
	}
}

type unavailableResourceAdmitter struct{}

func (unavailableResourceAdmitter) Admit(context.Context, resources.AdmissionRequest) (resources.AdmissionDecision, error) {
	return resources.AdmissionDecision{}, resources.ErrUnavailable
}

func TestE0911ScenarioCPostgresUnavailablePipelineIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	wirePipelineHooks(recorder)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("provider must not be called when admission storage fails")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	handler := e2eRoutedHandler(t, unavailableResourceAdmitter{}, upstream.URL)

	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "resource_admission_unavailable") {
		t.Fatalf("body = %s", rec.Body.String())
	}

	events := recorder.snapshot()
	assertSubsequence(t, events, "routing_selected", "resource_admission_started", "resource_admission_failed")
	assertAbsent(t, events, "execution_started", "provider_called", "resource_admission_accepted")
}

func TestE0911ScenarioDIdempotencyIntegration(t *testing.T) {
	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	ctx := context.Background()
	if _, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 3, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	req := resources.AdmissionRequest{ID: "e2e-idem-1", ProviderID: providers.OpenCodeGoID, ModelID: "pipeline-model"}
	if _, err := service.Admit(ctx, req); err != nil {
		t.Fatalf("first Admit: %v", err)
	}
	if _, err := service.Admit(ctx, req); err != nil {
		t.Fatalf("idempotent Admit: %v", err)
	}
	newID, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: newID, ProviderID: providers.OpenCodeGoID, ModelID: "pipeline-model"}); err != nil {
		t.Fatalf("new id Admit: %v", err)
	}

	period, _ := resources.UsagePeriodBounds(now)
	if count := globalCounter(t, pool, period); count != 2 {
		t.Fatalf("global counter = %d, want 2 (one idempotent + one new id)", count)
	}
}

func TestE0911ScenarioEAtomicScopesIntegration(t *testing.T) {
	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	ctx := context.Background()
	model := "pipeline-model"
	for _, input := range []resources.NewPolicyInput{
		{Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 2, Enabled: true},
		{Scope: resources.ResourceScope{Kind: resources.ScopeProvider, ProviderID: providers.OpenCodeGoID}, DailyLimit: 2, Enabled: true},
		{Scope: resources.ResourceScope{Kind: resources.ScopeModel, ProviderID: providers.OpenCodeGoID, ModelID: model}, DailyLimit: 1, Enabled: true},
	} {
		if _, err := service.CreatePolicy(ctx, input); err != nil {
			t.Fatalf("CreatePolicy: %v", err)
		}
	}

	id1, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id1, ProviderID: providers.OpenCodeGoID, ModelID: model}); err != nil {
		t.Fatalf("first Admit: %v", err)
	}

	period, _ := resources.UsagePeriodBounds(now)
	readScope := func(scope string) int {
		var count int
		_ = pool.QueryRow(ctx, `
SELECT c.admitted_count FROM resource_period_counters c
JOIN resource_policies p ON p.id = c.policy_id
WHERE p.scope = $1 AND c.period_start = $2`, scope, period).Scan(&count)
		return count
	}
	if readScope("global") != 1 || readScope("provider") != 1 || readScope("model") != 1 {
		t.Fatalf("counts after success global=%d provider=%d model=%d, want 1 each",
			readScope("global"), readScope("provider"), readScope("model"))
	}

	id2, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id2, ProviderID: providers.OpenCodeGoID, ModelID: model}); !resources.IsLimitExceeded(err) {
		t.Fatalf("second Admit = %v, want model limit exceeded", err)
	}
	if readScope("global") != 1 || readScope("provider") != 1 || readScope("model") != 1 {
		t.Fatalf("counts after reject global=%d provider=%d model=%d, want unchanged 1",
			readScope("global"), readScope("provider"), readScope("model"))
	}
}

func TestE0911ScenarioFNoApplicablePolicyPipelineIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	wirePipelineHooks(recorder)

	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	handler := e2eRoutedHandler(t, service, upstream.URL)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertSubsequence(t, recorder.snapshot(),
		"routing_selected", "resource_admission_accepted", "execution_started", "provider_called")

	var counterRows int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM resource_period_counters`).Scan(&counterRows); err != nil || counterRows != 0 {
		t.Fatalf("resource_period_counters rows = %d, err %v; want 0 without policies", counterRows, err)
	}
}

func TestE0911ScenarioGFileModeWithoutResourceAdmissionIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	wirePipelineHooks(recorder)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newEnabledClient(t, upstream.URL)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, client)
	manager := execution.NewMultiProviderManager(logger, func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
		if gateway.IntegrationTestHooks.OnExecutionStarted != nil {
			gateway.IntegrationTestHooks.OnExecutionStarted(target.ProviderID, target.ModelID)
		}
		resp, err := registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		if err == nil && gateway.IntegrationTestHooks.OnProviderCalled != nil {
			gateway.IntegrationTestHooks.OnProviderCalled(target.ProviderID, target.ModelID)
		}
		return resp, err
	}, time.Minute)
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, nil, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"m1","input":"hi"}`))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	events := recorder.snapshot()
	assertAbsent(t, events,
		"resource_admission_started", "resource_admission_accepted", "resource_admission_rejected", "resource_admission_failed")
	assertSubsequence(t, events, "execution_started", "provider_called")
}
