package gateway_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"aihub/internal/concurrency"
	"aihub/internal/gateway"
	"aihub/internal/resources"
	resourcespg "aihub/internal/resources/pgstore"
)

func TestE0931ScenarioAPipelineOrderIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	wireE0931PipelineHooks(recorder)

	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	resourceService := resources.NewService(resourcespg.New(pool), resources.WithClock(func() time.Time { return now }))
	concurrencyService := e0931ConcurrencyService(t, pool)
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	handler := newE0931PipelineHandler(t, pool, resourceService, concurrencyService, upstream.URL, time.Minute)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	assertSubsequence(t, recorder.snapshot(),
		"routing_selected",
		"concurrency_acquired",
		"resource_admission_accepted",
		"execution_started",
		"provider_called",
	)
}

func TestE0931ScenarioBConcurrencyRejectPreservesDailyQuotaIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	recorder := &pipelineEventRecorder{}
	var recordSecond atomic.Bool
	wireE0931PipelineHooks(recorder)
	gateway.IntegrationTestHooks.OnRoutingSelected = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("routing_selected")
		}
	}
	gateway.IntegrationTestHooks.OnConcurrencyAcquireAccepted = func(_, _, _ string) {
		if recordSecond.Load() {
			recorder.record("concurrency_acquired")
		}
	}
	prevRejected := gateway.IntegrationTestHooks.OnConcurrencyAcquireRejected
	gateway.IntegrationTestHooks.OnConcurrencyAcquireRejected = func(a, b, c string) {
		if recordSecond.Load() {
			recorder.record("concurrency_rejected")
		}
		if prevRejected != nil {
			prevRejected(a, b, c)
		}
	}
	gateway.IntegrationTestHooks.OnResourceAdmissionAccepted = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("resource_admission_accepted")
		}
	}
	gateway.IntegrationTestHooks.OnExecutionStarted = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("execution_started")
		}
	}
	gateway.IntegrationTestHooks.OnProviderCalled = func(_, _ string) {
		if recordSecond.Load() {
			recorder.record("provider_called")
		}
	}

	pool := e2eMigratedPool(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	resourceService := resources.NewService(resourcespg.New(pool), resources.WithClock(func() time.Time { return now }))
	concurrencyService := e0931ConcurrencyService(t, pool)
	ctx := context.Background()

	if _, err := resourceService.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 5, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if _, _, err := concurrencyService.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	releaseFirst := make(chan struct{})
	blockFirst := make(chan struct{})
	var heldFirst atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if heldFirst.CompareAndSwap(false, true) {
			close(blockFirst)
			<-releaseFirst
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	handler := newE0931PipelineHandler(t, pool, resourceService, concurrencyService, upstream.URL, time.Minute)

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
		t.Fatalf("first status = %d", code)
	}

	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d body = %s", second.Code, second.Body.String())
	}
	if strings.Contains(second.Body.String(), "concurrency_limit_exceeded") {
		t.Fatalf("body = %s", second.Body.String())
	}

	events := recorder.snapshot()
	assertSubsequence(t, events, "routing_selected", "concurrency_rejected", "concurrency_acquired")

	period, _ := resources.UsagePeriodBounds(now)
	if count := globalCounter(t, pool, period); count != 2 {
		t.Fatalf("daily counter = %d, want 2", count)
	}
}
