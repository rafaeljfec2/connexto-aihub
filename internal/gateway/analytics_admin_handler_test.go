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

	"aihub/internal/access"
	"aihub/internal/analytics"
	"aihub/internal/gateway"
	"aihub/internal/resources"
)

type analyticsMemoryRepo struct {
	total                          int
	admissionProviders             []analytics.ProviderDailyCount
	admissionModels                []analytics.ModelDailyCount
	admissionDistributionQuery     analytics.RangeQuery
	usageSummary                   analytics.ProviderUsageSummary
	usageDaily                     []analytics.ProviderUsageDailyTokens
	usageProviders                 []analytics.ProviderUsageProviderDaily
	usageModels                    []analytics.ProviderUsageModelDaily
	providerUsageDistributionQuery analytics.RangeQuery
}

func (r *analyticsMemoryRepo) CountAdmitted(context.Context, analytics.RangeQuery) (int, error) {
	return r.total, nil
}
func (r *analyticsMemoryRepo) ListDailyAdmissions(_ context.Context, query analytics.RangeQuery) ([]analytics.DailyAdmissionCount, error) {
	return []analytics.DailyAdmissionCount{{PeriodStart: query.From, AdmittedCount: r.total}}, nil
}
func (r *analyticsMemoryRepo) ListProviderDaily(_ context.Context, query analytics.RangeQuery) ([]analytics.ProviderDailyCount, error) {
	r.admissionDistributionQuery = query
	return r.admissionProviders, nil
}
func (r *analyticsMemoryRepo) ListModelDaily(_ context.Context, query analytics.RangeQuery) ([]analytics.ModelDailyCount, error) {
	r.admissionDistributionQuery = query
	return r.admissionModels, nil
}
func (r *analyticsMemoryRepo) ProviderUsageSummary(context.Context, analytics.RangeQuery) (analytics.ProviderUsageSummary, error) {
	return r.usageSummary, nil
}
func (r *analyticsMemoryRepo) ProviderUsageDaily(context.Context, analytics.RangeQuery) ([]analytics.ProviderUsageDailyTokens, error) {
	return r.usageDaily, nil
}
func (r *analyticsMemoryRepo) ProviderUsageByProvider(_ context.Context, query analytics.RangeQuery) ([]analytics.ProviderUsageProviderDaily, error) {
	r.providerUsageDistributionQuery = query
	return r.usageProviders, nil
}
func (r *analyticsMemoryRepo) ProviderUsageByModel(_ context.Context, query analytics.RangeQuery) ([]analytics.ProviderUsageModelDaily, error) {
	r.providerUsageDistributionQuery = query
	return r.usageModels, nil
}

func analyticsAdminServer(t *testing.T, total int) *httptest.Server {
	t.Helper()
	return analyticsAdminServerWithRepo(t, &analyticsMemoryRepo{total: total})
}

func analyticsAdminServerWithRepo(t *testing.T, memoryRepo *analyticsMemoryRepo) *httptest.Server {
	t.Helper()
	repo := &inMemoryResources{policies: map[int64]resources.ResourcePolicy{}}
	resourcesService := resources.NewService(repo)
	analyticsService := analytics.NewService(memoryRepo)
	admin := gateway.NewAnalyticsAdmin(analyticsService, resourcesService, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

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
	mux.Handle("GET /api/resource-analytics/provider-usage", admin.ProviderUsage())
	mux.Handle("GET /api/resource-analytics/provider-usage/distribution", admin.ProviderUsageDistribution())
	server := httptest.NewServer(policy.Authorize(mux))
	t.Cleanup(server.Close)
	return server
}

func TestProviderUsageHTTPPreservesUnknownAndExplicitZero(t *testing.T) {
	zero := int64(0)
	day := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	repo := &analyticsMemoryRepo{
		usageSummary: analytics.ProviderUsageSummary{OutputTokens: &zero},
		usageDaily:   []analytics.ProviderUsageDailyTokens{{PeriodStart: day, OutputTokens: &zero}},
	}
	server := analyticsAdminServerWithRepo(t, repo)
	url := server.URL + "/api/resource-analytics/provider-usage?from=2026-11-03&to=2026-11-03"
	status, payload := resourceAdminCall(t, http.MethodGet, url, "", resourceAdminToken)
	if status != http.StatusOK {
		t.Fatalf("status=%d payload=%s", status, payload)
	}
	var response struct {
		InputTokens  json.RawMessage `json:"input_tokens"`
		OutputTokens json.RawMessage `json:"output_tokens"`
		TotalTokens  json.RawMessage `json:"total_tokens"`
		Daily        []struct {
			InputTokens  json.RawMessage `json:"input_tokens"`
			OutputTokens json.RawMessage `json:"output_tokens"`
			TotalTokens  json.RawMessage `json:"total_tokens"`
		} `json:"daily"`
	}
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if string(response.InputTokens) != "null" || string(response.TotalTokens) != "null" || string(response.OutputTokens) != "0" {
		t.Fatalf("summary tokens = input %s, output %s, total %s", response.InputTokens, response.OutputTokens, response.TotalTokens)
	}
	if len(response.Daily) != 1 || string(response.Daily[0].InputTokens) != "null" || string(response.Daily[0].OutputTokens) != "0" || string(response.Daily[0].TotalTokens) != "null" {
		t.Fatalf("daily tokens = %+v", response.Daily)
	}
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
