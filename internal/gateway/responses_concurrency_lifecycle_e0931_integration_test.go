package gateway_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/concurrency"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/resources"
	resourcespg "aihub/internal/resources/pgstore"
)

func e0931LeaseSetup(t *testing.T) (*resources.Service, *concurrency.Service, *httptest.Server, func(time.Time) int) {
	t.Helper()
	pool := e2eMigratedPool(t)
	now := time.Now().UTC()
	resourceService := resources.NewService(resourcespg.New(pool), resources.WithClock(func() time.Time { return now }))
	concurrencyService := e0931ConcurrencyService(t, pool)
	ctx := context.Background()
	if _, err := resourceService.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 100, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if _, _, err := concurrencyService.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)
	countFn := func(at time.Time) int { return countActiveConcurrencyLeases(t, pool, at) }
	return resourceService, concurrencyService, upstream, countFn
}

func TestE0931ScenarioE01NormalCompletionReleasesLeaseIntegration(t *testing.T) {
	resourceService, concurrencyService, upstream, countLeases := e0931LeaseSetup(t)
	handler := newE0931PipelineHandler(t, nil, resourceService, concurrencyService, upstream.URL, time.Minute)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if countLeases(time.Now().UTC()) != 0 {
		t.Fatalf("active leases after success = %d, want 0", countLeases(time.Now().UTC()))
	}
}

func TestE0931ScenarioE02ProviderOpenFailureReleasesLeaseIntegration(t *testing.T) {
	resourceService, concurrencyService, upstream, countLeases := e0931LeaseSetup(t)
	upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	handler := newE0931PipelineHandler(t, nil, resourceService, concurrencyService, upstream.URL, time.Minute)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if countLeases(time.Now().UTC()) != 0 {
		t.Fatalf("active leases after provider failure = %d, want 0", countLeases(time.Now().UTC()))
	}
}

func TestE0931ScenarioE03OpeningTimeoutReleasesLeaseIntegration(t *testing.T) {
	resourceService, concurrencyService, upstream, countLeases := e0931LeaseSetup(t)
	upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		}
	})
	transport := &http.Transport{ResponseHeaderTimeout: 40 * time.Millisecond}
	client, err := providers.NewClient(config.ProviderOpenCodeGo{
		BaseURL: upstream.URL,
		APIKey:  providerCredential,
	}, &http.Client{Transport: transport})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(transport.CloseIdleConnections)

	logger, _ := newTestLogger()
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
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		},
		time.Minute,
		execution.WithObserver(concurrency.NewReleaseObserver(concurrencyService, nil)),
	)
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}}
		},
		StaleAfter: 24 * time.Hour,
	}, resourceService, concurrencyService, nil)

	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if countLeases(time.Now().UTC()) != 0 {
		t.Fatalf("active leases after timeout = %d, want 0", countLeases(time.Now().UTC()))
	}
}

func TestE0931ScenarioE04ClientCancelReleasesLeaseIntegration(t *testing.T) {
	resourceService, concurrencyService, upstream, countLeases := e0931LeaseSetup(t)
	block := make(chan struct{})
	upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-block:
			w.WriteHeader(http.StatusOK)
		}
	})
	handler := newE0931PipelineHandler(t, nil, resourceService, concurrencyService, upstream.URL, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		body := `{"input":[{"type":"input_text","text":"hello"}]}`
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not finish after client cancel")
	}
	close(block)
	if countLeases(time.Now().UTC()) != 0 {
		t.Fatalf("active leases after client cancel = %d, want 0", countLeases(time.Now().UTC()))
	}
}

func TestE0931ScenarioE05StreamInactivityReleasesLeaseIntegration(t *testing.T) {
	resourceService, concurrencyService, upstream, countLeases := e0931LeaseSetup(t)
	upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: chunk\n\n")
		flusher, ok := w.(http.Flusher)
		if ok {
			flusher.Flush()
		}
		time.Sleep(500 * time.Millisecond)
	})
	handler := newE0931PipelineHandler(t, nil, resourceService, concurrencyService, upstream.URL, 50*time.Millisecond)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusGatewayTimeout && rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if countLeases(time.Now().UTC()) != 0 {
		t.Fatalf("active leases after stream stall = %d, want 0", countLeases(time.Now().UTC()))
	}
}

func TestE0931ScenarioE06TransportFailureReleasesLeaseIntegration(t *testing.T) {
	resourceService, concurrencyService, _, countLeases := e0931LeaseSetup(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	upstream.Close()
	handler := newE0931PipelineHandler(t, nil, resourceService, concurrencyService, upstream.URL, time.Minute)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusBadGateway && rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if countLeases(time.Now().UTC()) != 0 {
		t.Fatalf("active leases after transport failure = %d, want 0", countLeases(time.Now().UTC()))
	}
}

func TestE0931ScenarioE07ShutdownReleasesLeaseIntegration(t *testing.T) {
	resourceService, concurrencyService, upstream, countLeases := e0931LeaseSetup(t)
	releaseUpstream := make(chan struct{})
	upstream.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-releaseUpstream:
			w.WriteHeader(http.StatusOK)
		}
	})
	logger, _ := newTestLogger()
	client := newEnabledClient(t, upstream.URL)
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
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		},
		time.Minute,
		execution.WithObserver(concurrency.NewReleaseObserver(concurrencyService, nil)),
	)
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}}
		},
		StaleAfter: 24 * time.Hour,
	}, resourceService, concurrencyService, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = postRoutedResponses(t, handler)
	}()
	time.Sleep(100 * time.Millisecond)
	manager.BeginDrain()
	close(releaseUpstream)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not finish after shutdown")
	}
	if countLeases(time.Now().UTC()) != 0 {
		t.Fatalf("active leases after shutdown = %d, want 0", countLeases(time.Now().UTC()))
	}
}
