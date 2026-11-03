package gateway_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/catalog"
	"aihub/internal/concurrency"
	"aihub/internal/concurrency/pgstore"
	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/resources"
	resourcespg "aihub/internal/resources/pgstore"
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
		Catalog:    catalogManager,
		Statuses:   func() []providers.Status { return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}} },
		StaleAfter: 24 * time.Hour,
	}, resourceService, concurrencyService)
}

func TestE093PipelineConcurrencyBeforeDailyQuotaIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	var recordSecond atomic.Bool
	wireConcurrencyPipelineHooks(recorder)
	prevStarted := gateway.IntegrationTestHooks.OnConcurrencyAcquireStarted
	prevAccepted := gateway.IntegrationTestHooks.OnConcurrencyAcquireAccepted
	prevRejected := gateway.IntegrationTestHooks.OnConcurrencyAcquireRejected
	prevFailed := gateway.IntegrationTestHooks.OnConcurrencyAcquireFailed
	prevExec := gateway.IntegrationTestHooks.OnExecutionStarted
	prevProvider := gateway.IntegrationTestHooks.OnProviderCalled
	gateway.IntegrationTestHooks.OnConcurrencyAcquireStarted = func(a, b, c string) {
		if recordSecond.Load() {
			prevStarted(a, b, c)
		}
	}
	gateway.IntegrationTestHooks.OnConcurrencyAcquireAccepted = func(a, b, c string) {
		if recordSecond.Load() {
			prevAccepted(a, b, c)
		}
	}
	gateway.IntegrationTestHooks.OnConcurrencyAcquireRejected = func(a, b, c string) {
		if recordSecond.Load() {
			prevRejected(a, b, c)
		}
	}
	gateway.IntegrationTestHooks.OnConcurrencyAcquireFailed = func(a, b, c string) {
		if recordSecond.Load() {
			prevFailed(a, b, c)
		}
	}
	gateway.IntegrationTestHooks.OnExecutionStarted = func(a, b string) {
		if recordSecond.Load() {
			prevExec(a, b)
		}
	}
	gateway.IntegrationTestHooks.OnProviderCalled = func(a, b string) {
		if recordSecond.Load() {
			prevProvider(a, b)
		}
	}
	gateway.IntegrationTestHooks.OnRoutingSelected = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("routing_selected")
		}
	}
	gateway.IntegrationTestHooks.OnResourceAdmissionStarted = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("resource_admission_started")
		}
	}
	gateway.IntegrationTestHooks.OnResourceAdmissionAccepted = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("resource_admission_accepted")
		}
	}
	gateway.IntegrationTestHooks.OnResourceAdmissionRejected = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("resource_admission_rejected")
		}
	}
	gateway.IntegrationTestHooks.OnResourceAdmissionFailed = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("resource_admission_failed")
		}
	}

	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	resourceService := resources.NewService(resourcespg.New(pool), resources.WithClock(func() time.Time { return now }))
	concurrencyService := concurrency.NewService(pgstore.New(pool))
	ctx := context.Background()

	if _, err := resourceService.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 5, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy resource: %v", err)
	}
	if _, _, err := concurrencyService.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy concurrency: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	releaseFirst := make(chan struct{})
	blockFirst := make(chan struct{})
	upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-releaseFirst:
		default:
			close(blockFirst)
			<-releaseFirst
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	})

	handler := e2eRoutedHandlerWithConcurrency(t, pool, resourceService, concurrencyService, upstream.URL)

	firstDone := make(chan int)
	go func() {
		rec := postRoutedResponses(t, handler)
		firstDone <- rec.Code
	}()
	<-blockFirst

	recorder.events = nil
	recordSecond.Store(true)
	second := postRoutedResponses(t, handler)
	recordSecond.Store(false)
	close(releaseFirst)
	if code := <-firstDone; code != http.StatusOK {
		t.Fatalf("first status = %d, want 200 while slot held", code)
	}
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d body = %s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Body.String(), "concurrency_limit_exceeded") {
		t.Fatalf("body = %s", second.Body.String())
	}

	events := recorder.snapshot()
	assertSubsequence(t, events,
		"routing_selected",
		"concurrency_acquire_started",
		"concurrency_acquire_rejected",
	)
	assertAbsent(t, events,
		"resource_admission_started",
		"resource_admission_accepted",
		"execution_started",
		"provider_called",
	)

	period, _ := resources.UsagePeriodBounds(now)
	if count := globalCounter(t, pool, period); count != 1 {
		t.Fatalf("daily counter = %d, want 1 (concurrency reject must not consume quota)", count)
	}
}

func TestE093PipelineHappyPathOrderIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	wireConcurrencyPipelineHooks(recorder)

	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	resourceService := resources.NewService(resourcespg.New(pool), resources.WithClock(func() time.Time { return now }))
	concurrencyService := concurrency.NewService(pgstore.New(pool))
	ctx := context.Background()

	if _, err := resourceService.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 5, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if _, _, err := concurrencyService.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 2, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	handler := e2eRoutedHandlerWithConcurrency(t, pool, resourceService, concurrencyService, upstream.URL)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	assertSubsequence(t, recorder.snapshot(),
		"routing_selected",
		"concurrency_acquire_started",
		"concurrency_acquire_accepted",
		"resource_admission_started",
		"resource_admission_accepted",
		"execution_started",
		"provider_called",
	)
}

func TestE093TwoServicesShareConcurrencyLimitIntegration(t *testing.T) {
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	store := pgstore.New(pool)
	serviceA := concurrency.NewService(store)
	serviceB := concurrency.NewService(store)
	ctx := context.Background()

	if _, _, err := serviceA.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	if _, err := serviceA.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "shared-exec-1", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("service A acquire: %v", err)
	}
	if _, err := serviceB.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "shared-exec-2", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); !concurrency.IsLimitExceeded(err) {
		t.Fatalf("service B acquire = %v, want limit exceeded", err)
	}
}
