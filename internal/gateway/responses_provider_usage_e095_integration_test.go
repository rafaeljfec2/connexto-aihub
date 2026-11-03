package gateway_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/analytics"
	analyticspg "aihub/internal/analytics/pgstore"
	"aihub/internal/catalog"
	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/metering"
	meteringpg "aihub/internal/metering/pgstore"
	"aihub/internal/providers"
	"aihub/internal/resources"
	"aihub/internal/resources/pgstore"
)

func TestE095ProviderUsagePipelineIntegration(t *testing.T) {
	resetIntegrationHooks(t)
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	resourceService := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	ctx := context.Background()
	if _, err := resourceService.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 10, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, strings.Join([]string{
			"event: response.created\ndata: {\"type\":\"response.created\"}\n\n",
			"event: response.completed\ndata: {\"usage\":{\"input_tokens\":4,\"output_tokens\":6,\"total_tokens\":10}}\n\n",
		}, ""))
	}))
	t.Cleanup(upstream.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newEnabledClient(t, upstream.URL)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, client)
	catalogManager := gatewayTestCatalog(t, now, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{gatewayTestDescriptor("usage-model")}, nil
		},
	})
	manager := execution.NewMultiProviderManager(logger, func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
		return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
	}, time.Minute)
	usageService := metering.NewService(meteringpg.New(pool), metering.WithLogger(logger))
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}}
		},
		StaleAfter: 24 * time.Hour,
	}, resourceService, nil, usageService)

	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var executionID string
	if err := pool.QueryRow(ctx, `
SELECT execution_id FROM provider_execution_usage ORDER BY recorded_at DESC LIMIT 1`).Scan(&executionID); err != nil {
		t.Fatalf("lookup usage row: %v", err)
	}
	store := meteringpg.New(pool)
	got, err := store.Get(ctx, executionID)
	if err != nil {
		t.Fatalf("Get usage: %v", err)
	}
	if got.Status != metering.StatusReported || got.InputTokens == nil || *got.InputTokens != 4 {
		t.Fatalf("usage row = %+v", got)
	}

	analyticsService := analytics.NewService(analyticspg.New(pool))
	report, err := analyticsService.ProviderUsage(ctx, analytics.RangeQuery{From: now, To: now})
	if err != nil {
		t.Fatalf("ProviderUsage: %v", err)
	}
	if report.InputTokens != 4 || report.ReportedExecutions != 1 {
		t.Fatalf("analytics = %+v", report)
	}
}

func TestE095ResponseWithoutUsageStillSucceeds(t *testing.T) {
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "event: response.completed\ndata: {}\n\n")
	}))
	t.Cleanup(upstream.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newEnabledClient(t, upstream.URL)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, client)
	manager := execution.NewMultiProviderManager(logger, func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
		return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
	}, time.Minute)
	usageService := metering.NewService(meteringpg.New(pool), metering.WithLogger(logger))
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, nil, nil, nil, usageService)
	rec := postRoutedResponses(t, handler)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var status string
	if err := pool.QueryRow(context.Background(), `
SELECT usage_status FROM provider_execution_usage ORDER BY recorded_at DESC LIMIT 1`).Scan(&status); err != nil {
		t.Fatalf("lookup usage row: %v", err)
	}
	if status != string(metering.StatusUnavailable) {
		t.Fatalf("usage_status = %s", status)
	}
}

func TestE095AnalyticsEndpointReturnsProviderUsage(t *testing.T) {
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	now := time.Date(2026, 11, 3, 16, 0, 0, 0, time.UTC)
	store := meteringpg.New(pool)
	_, err := store.Insert(context.Background(), metering.Record{
		ExecutionID:  "exec-analytics-1",
		ProviderID:   providers.OpenCodeGoID,
		ModelID:      "m1",
		InputTokens:  int64Ref(7),
		OutputTokens: int64Ref(3),
		TotalTokens:  int64Ref(10),
		Status:       metering.StatusReported,
		Source:       metering.SourceOpenCodeGo,
		RecordedAt:   now,
	})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	analyticsService := analytics.NewService(analyticspg.New(pool))
	admin := gateway.NewAnalyticsAdmin(analyticsService, resources.NewService(pgstore.New(pool)), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	mux.Handle("GET /api/resource-analytics/provider-usage", admin.ProviderUsage())
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/api/resource-analytics/provider-usage?from=2026-11-03&to=2026-11-03")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["input_tokens"].(float64) != 7 {
		t.Fatalf("payload = %#v", payload)
	}
}

func int64Ref(v int64) *int64 {
	return &v
}
