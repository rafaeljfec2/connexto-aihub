package gateway_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/access"
	"aihub/internal/analytics"
	analyticspg "aihub/internal/analytics/pgstore"
	"aihub/internal/concurrency"
	concurrencypg "aihub/internal/concurrency/pgstore"
	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/resources"
	"aihub/internal/resources/pgstore"
)

func analyticsPGServer(t *testing.T, pool *pgxpool.Pool) *httptest.Server {
	t.Helper()
	resourceService := resources.NewService(pgstore.New(pool))
	concurrencyService := concurrency.NewService(concurrencypg.New(pool))
	analyticsService := analytics.NewService(analyticspg.New(pool))
	admin := gateway.NewAnalyticsAdmin(analyticsService, resourceService, concurrencyService, slog.New(slog.NewTextHandler(io.Discard, nil)))

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey: resourceGatewayToken, AdminAPIKey: resourceAdminToken, MaxBodyBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/resource-analytics/admissions", admin.Admissions())
	mux.Handle("GET /api/resource-analytics/admissions/distribution", admin.Distribution())
	mux.Handle("GET /api/resource-analytics/quota-utilization", admin.QuotaUtilization())
	mux.Handle("GET /api/resource-analytics/concurrency", admin.Concurrency())
	mux.Handle("GET /api/resource-analytics/provider-usage", admin.ProviderUsage())
	mux.Handle("GET /api/resource-analytics/provider-usage/distribution", admin.ProviderUsageDistribution())
	server := httptest.NewServer(policy.Authorize(mux))
	t.Cleanup(server.Close)
	return server
}

func TestAnalyticsAdminPGIntegration(t *testing.T) {
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	day := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	resourceService := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time {
		return day.Add(10 * time.Hour)
	}))
	ctx := context.Background()
	if _, err := resourceService.Admit(ctx, resources.AdmissionRequest{
		ID: "adm-analytics-1", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if _, err := resourceService.Admit(ctx, resources.AdmissionRequest{
		ID: "adm-analytics-2", ProviderID: providers.OpenRouterID, ModelID: "m2",
	}); err != nil {
		t.Fatalf("Admit second provider: %v", err)
	}

	server := analyticsPGServer(t, pool)
	from := day.Format(time.DateOnly)
	status, payload := resourceAdminCall(t, http.MethodGet,
		server.URL+"/api/resource-analytics/admissions?from="+from+"&to="+from, "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"total_admitted":2`) {
		t.Fatalf("admissions status=%d payload=%s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodGet,
		server.URL+"/api/resource-analytics/admissions/distribution?from="+from+"&to="+from, "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"by_provider"`) || !strings.Contains(payload, providers.OpenCodeGoID) {
		t.Fatalf("distribution status=%d payload=%s", status, payload)
	}
	status, payload = resourceAdminCall(t, http.MethodGet,
		server.URL+"/api/resource-analytics/admissions/distribution?from="+from+"&to="+from+"&provider_id="+providers.OpenCodeGoID, "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"admitted_count":1`) || strings.Contains(payload, providers.OpenRouterID) {
		t.Fatalf("provider-filtered distribution status=%d payload=%s", status, payload)
	}
	status, payload = resourceAdminCall(t, http.MethodGet,
		server.URL+"/api/resource-analytics/admissions/distribution?from="+from+"&to="+from+"&provider_id="+providers.OpenRouterID+"&model_id=m2", "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"model_id":"m2"`) || strings.Contains(payload, providers.OpenCodeGoID) {
		t.Fatalf("combined-filter distribution status=%d payload=%s", status, payload)
	}
	status, _ = resourceAdminCall(t, http.MethodGet,
		server.URL+"/api/resource-analytics/admissions/distribution?from="+from+"&to="+from+"&model_id=m1", "", resourceAdminToken)
	if status != http.StatusBadRequest {
		t.Fatalf("model-only distribution status=%d, want 400", status)
	}
	status, payload = resourceAdminCall(t, http.MethodGet,
		server.URL+"/api/resource-analytics/admissions/distribution?from="+from+"&to="+from+"&provider_id=missing", "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"by_provider":[]`) || !strings.Contains(payload, `"by_model":[]`) {
		t.Fatalf("empty filtered distribution status=%d payload=%s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodGet, server.URL+"/api/resource-analytics/concurrency", "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"concurrency"`) {
		t.Fatalf("concurrency status=%d payload=%s", status, payload)
	}
}
