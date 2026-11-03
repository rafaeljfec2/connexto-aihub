package gateway_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/access"
	"aihub/internal/analytics"
	"aihub/internal/gateway"
	"aihub/internal/resources"
)

type analyticsMemoryRepo struct {
	total int
}

func (r *analyticsMemoryRepo) CountAdmitted(context.Context, analytics.RangeQuery) (int, error) {
	return r.total, nil
}
func (r *analyticsMemoryRepo) ListDailyAdmissions(_ context.Context, query analytics.RangeQuery) ([]analytics.DailyAdmissionCount, error) {
	return []analytics.DailyAdmissionCount{{PeriodStart: query.From, AdmittedCount: r.total}}, nil
}
func (r *analyticsMemoryRepo) ListProviderDaily(context.Context, analytics.RangeQuery) ([]analytics.ProviderDailyCount, error) {
	return nil, nil
}
func (r *analyticsMemoryRepo) ListModelDaily(context.Context, analytics.RangeQuery) ([]analytics.ModelDailyCount, error) {
	return nil, nil
}

func analyticsAdminServer(t *testing.T, total int) *httptest.Server {
	t.Helper()
	repo := &inMemoryResources{policies: map[int64]resources.ResourcePolicy{}}
	resourcesService := resources.NewService(repo)
	analyticsService := analytics.NewService(&analyticsMemoryRepo{total: total})
	admin := gateway.NewAnalyticsAdmin(analyticsService, resourcesService, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey: resourceGatewayToken, AdminAPIKey: resourceAdminToken, MaxBodyBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/resource-analytics/admissions", admin.Admissions())
	mux.Handle("GET /api/resource-analytics/quota-utilization", admin.QuotaUtilization())
	server := httptest.NewServer(policy.Authorize(mux))
	t.Cleanup(server.Close)
	return server
}

func TestAnalyticsAdminRequiresRange(t *testing.T) {
	server := analyticsAdminServer(t, 0)
	status, _ := resourceAdminCall(t, http.MethodGet, server.URL+"/api/resource-analytics/admissions", "", resourceAdminToken)
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
}

func TestAnalyticsAdmissionsHappyPath(t *testing.T) {
	server := analyticsAdminServer(t, 2)
	url := server.URL + "/api/resource-analytics/admissions?from=2026-11-01&to=2026-11-02"
	status, payload := resourceAdminCall(t, http.MethodGet, url, "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"total_admitted":2`) {
		t.Fatalf("status=%d payload=%s", status, payload)
	}
	if !strings.Contains(payload, `"admitted_count":0`) {
		t.Fatalf("expected gap-filled zero day, payload=%s", payload)
	}
}

func TestAnalyticsQuotaUtilizationIncludesPercent(t *testing.T) {
	repo := &inMemoryResources{policies: map[int64]resources.ResourcePolicy{}}
	resourcesService := resources.NewService(repo)
	if _, err := resourcesService.CreatePolicy(context.Background(), resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 10, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	analyticsService := analytics.NewService(&analyticsMemoryRepo{})
	admin := gateway.NewAnalyticsAdmin(analyticsService, resourcesService, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey: resourceGatewayToken, AdminAPIKey: resourceAdminToken, MaxBodyBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/resource-analytics/quota-utilization", admin.QuotaUtilization())
	server := httptest.NewServer(policy.Authorize(mux))
	t.Cleanup(server.Close)

	status, payload := resourceAdminCall(t, http.MethodGet, server.URL+"/api/resource-analytics/quota-utilization", "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"utilization_percent":0`) {
		t.Fatalf("status=%d payload=%s", status, payload)
	}
}
