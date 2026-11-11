package gateway_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"aihub/internal/analytics"
)

func TestAnalyticsAdmissionDistributionHTTPFilters(t *testing.T) {
	day := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	repo := &analyticsMemoryRepo{
		admissionProviders: []analytics.ProviderDailyCount{{
			PeriodStart: day, ProviderID: "opencode-go", AdmittedCount: 2,
		}},
		admissionModels: []analytics.ModelDailyCount{{
			PeriodStart: day, ProviderID: "opencode-go", ModelID: "m1", AdmittedCount: 2,
		}},
	}
	server := analyticsAdminServerWithRepo(t, repo)
	baseURL := server.URL + "/api/resource-analytics/admissions/distribution?from=2026-11-03&to=2026-11-03"
	status, payload := resourceAdminCall(t, http.MethodGet, baseURL+"&provider_id=opencode-go", "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"provider_id":"opencode-go"`) {
		t.Fatalf("provider filter status=%d payload=%s", status, payload)
	}
	if repo.admissionDistributionQuery.ProviderID != "opencode-go" || repo.admissionDistributionQuery.ModelID != "" {
		t.Fatalf("provider query = %+v", repo.admissionDistributionQuery)
	}

	status, payload = resourceAdminCall(t, http.MethodGet, baseURL+"&provider_id=opencode-go&model_id="+url.QueryEscape("m1"), "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"model_id":"m1"`) {
		t.Fatalf("combined filter status=%d payload=%s", status, payload)
	}
	if repo.admissionDistributionQuery.ProviderID != "opencode-go" || repo.admissionDistributionQuery.ModelID != "m1" {
		t.Fatalf("combined query = %+v", repo.admissionDistributionQuery)
	}
	status, payload = resourceAdminCall(t, http.MethodGet, baseURL+"&provider_id=opencode-go", "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"provider_id":"opencode-go"`) {
		t.Fatalf("provider-only filter status=%d payload=%s", status, payload)
	}
	if repo.admissionDistributionQuery.ProviderID != "opencode-go" || repo.admissionDistributionQuery.ModelID != "" {
		t.Fatalf("provider-only query = %+v", repo.admissionDistributionQuery)
	}
}

func TestAnalyticsAdmissionDistributionRejectsModelOnlyAndReturnsEmptyResults(t *testing.T) {
	server := analyticsAdminServerWithRepo(t, &analyticsMemoryRepo{})
	baseURL := server.URL + "/api/resource-analytics/admissions/distribution?from=2026-11-03&to=2026-11-03"
	status, _ := resourceAdminCall(t, http.MethodGet, baseURL+"&model_id=m1", "", resourceAdminToken)
	if status != http.StatusBadRequest {
		t.Fatalf("model-only filter status = %d, want 400", status)
	}
	status, payload := resourceAdminCall(t, http.MethodGet, baseURL+"&provider_id=missing", "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"by_provider":[]`) || !strings.Contains(payload, `"by_model":[]`) {
		t.Fatalf("empty distribution status=%d payload=%s", status, payload)
	}
}

func TestProviderUsageDistributionHTTPFiltersAndPreservesNull(t *testing.T) {
	day := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	zero := int64(0)
	repo := &analyticsMemoryRepo{
		usageProviders: []analytics.ProviderUsageProviderDaily{{
			PeriodStart: day, ProviderID: "opencode-go", OutputTokens: &zero, Executions: 1,
		}},
		usageModels: []analytics.ProviderUsageModelDaily{{
			PeriodStart: day, ProviderID: "opencode-go", ModelID: "m1", OutputTokens: &zero, Executions: 1,
		}},
	}
	server := analyticsAdminServerWithRepo(t, repo)
	url := server.URL + "/api/resource-analytics/provider-usage/distribution?from=2026-11-03&to=2026-11-03&provider_id=opencode-go&model_id=m1"
	status, payload := resourceAdminCall(t, http.MethodGet, url, "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"input_tokens":null`) || !strings.Contains(payload, `"output_tokens":0`) || !strings.Contains(payload, `"model_id":"m1"`) {
		t.Fatalf("distribution status=%d payload=%s", status, payload)
	}
	if repo.providerUsageDistributionQuery.ProviderID != "opencode-go" || repo.providerUsageDistributionQuery.ModelID != "m1" {
		t.Fatalf("distribution query = %+v", repo.providerUsageDistributionQuery)
	}
	url = server.URL + "/api/resource-analytics/provider-usage/distribution?from=2026-11-03&to=2026-11-03&provider_id=opencode-go"
	status, payload = resourceAdminCall(t, http.MethodGet, url, "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"provider_id":"opencode-go"`) {
		t.Fatalf("provider-only usage distribution status=%d payload=%s", status, payload)
	}
	if repo.providerUsageDistributionQuery.ProviderID != "opencode-go" || repo.providerUsageDistributionQuery.ModelID != "" {
		t.Fatalf("provider-only usage query = %+v", repo.providerUsageDistributionQuery)
	}

	emptyServer := analyticsAdminServerWithRepo(t, &analyticsMemoryRepo{})
	url = emptyServer.URL + "/api/resource-analytics/provider-usage/distribution?from=2026-11-03&to=2026-11-03"
	status, payload = resourceAdminCall(t, http.MethodGet, url, "", resourceAdminToken)
	if status != http.StatusOK || !strings.Contains(payload, `"by_provider":[]`) || !strings.Contains(payload, `"by_model":[]`) {
		t.Fatalf("empty distribution status=%d payload=%s", status, payload)
	}
}
