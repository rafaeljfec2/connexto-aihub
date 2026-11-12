package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMapCursorUsageMetrics_mapsUsdTotals(t *testing.T) {
	usage := map[string]interface{}{
		"total_usage":              41.68,
		"total_credits":            46.15,
		"weekly_used_percent":      90.31,
		"prepaid_available_credits": 3.15,
		"currency":                 "USD",
	}
	metrics := mapCursorUsageMetrics(usage)

	usageMetric, ok := findQuotaMetric(metrics, "total_usage")
	if !ok || usageMetric.Used == nil || *usageMetric.Used != 41.68 {
		t.Fatalf("total_usage metric = %+v, want used 41.68", usageMetric)
	}
	creditsMetric, ok := findQuotaMetric(metrics, "total_credits")
	if !ok || creditsMetric.Limit == nil || *creditsMetric.Limit != 46.15 {
		t.Fatalf("total_credits metric = %+v, want limit 46.15", creditsMetric)
	}
	if creditsMetric.ConsumedPercent == nil || *creditsMetric.ConsumedPercent != 90.31 {
		t.Fatalf("total_credits ConsumedPercent = %v, want 90.31", creditsMetric.ConsumedPercent)
	}
	percentMetric, ok := findQuotaMetric(metrics, "weekly_used_percent")
	if !ok || percentMetric.ConsumedPercent == nil || *percentMetric.ConsumedPercent != 90.31 {
		t.Fatalf("weekly_used_percent metric = %+v, want 90.31", percentMetric)
	}
	prepaidMetric, ok := findQuotaMetric(metrics, "prepaid_available_credits")
	if !ok || prepaidMetric.Remaining == nil || *prepaidMetric.Remaining != 3.15 {
		t.Fatalf("prepaid_available_credits metric = %+v, want 3.15", prepaidMetric)
	}
}

func TestMapCursorUsageMetrics_mapsExpandedUsageStatusFields(t *testing.T) {
	usage := map[string]interface{}{
		"usage_availability":           "available",
		"weekly_included_status":       "metered",
		"total_usage":                  0.18,
		"total_credits":                161.59,
		"weekly_period_start":          "2026-11-09T21:45:07.488Z",
		"weekly_resets_at":             "2026-11-16T21:45:07.488Z",
		"weekly_used_percent":          0.1113930317,
		"prepaid_available_credits":    0.79,
		"monthly_additional_usage":     9.22,
		"monthly_credits_used":         0.0,
		"monthly_additional_usage_off": false,
		"monthly_period_start":         "2026-11-01T00:00:00.000Z",
		"monthly_resets_at":            "2026-12-01T00:00:00.000Z",
		"weekly_usage_by_product": []interface{}{
			map[string]interface{}{
				"product":      "cursor",
				"used_cents":   18.0,
				"used_usd":     0.18,
				"used_percent": 0.1113930317,
			},
		},
	}

	metrics := mapCursorUsageMetrics(usage)

	creditsMetric, ok := findQuotaMetric(metrics, "total_credits")
	if !ok || creditsMetric.StartsAt == nil || *creditsMetric.StartsAt != "2026-11-09T21:45:07.488Z" {
		t.Fatalf("total_credits StartsAt = %+v, want weekly_period_start", creditsMetric)
	}
	statusMetric, ok := findQuotaMetric(metrics, "weekly_included_status")
	if !ok || statusMetric.Scope != "weekly_status:metered" {
		t.Fatalf("weekly_included_status = %+v, want scope weekly_status:metered", statusMetric)
	}
	productMetric, ok := findQuotaMetric(metrics, "weekly_product_cursor")
	if !ok || productMetric.Used == nil || *productMetric.Used != 0.18 || productMetric.Scope != "product:cursor" {
		t.Fatalf("weekly_product_cursor = %+v", productMetric)
	}
	monthlyMetric, ok := findQuotaMetric(metrics, "monthly_additional_usage")
	if !ok || monthlyMetric.Used == nil || *monthlyMetric.Used != 9.22 || monthlyMetric.Limit != nil {
		t.Fatalf("monthly_additional_usage = %+v, want used 9.22 and nil limit", monthlyMetric)
	}
	if monthlyMetric.ResetsAt == nil || *monthlyMetric.ResetsAt != "2026-12-01T00:00:00.000Z" {
		t.Fatalf("monthly_additional_usage ResetsAt = %+v", monthlyMetric.ResetsAt)
	}
	if monthlyMetric.Scope != "additional_enabled" {
		t.Fatalf("monthly_additional_usage Scope = %q, want additional_enabled", monthlyMetric.Scope)
	}
	creditsUsed, ok := findQuotaMetric(metrics, "monthly_credits_used")
	if !ok || creditsUsed.Used == nil || *creditsUsed.Used != 0 {
		t.Fatalf("monthly_credits_used = %+v, want used 0", creditsUsed)
	}
}

func findQuotaMetric(metrics []ProviderQuotaMetric, name string) (ProviderQuotaMetric, bool) {
	for _, metric := range metrics {
		if metric.Metric == name {
			return metric, true
		}
	}
	return ProviderQuotaMetric{}, false
}

func TestCursorUsageQuotaAdapter_slowCollectWithinDedicatedTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "usage/collect") {
			http.NotFound(w, r)
			return
		}
		time.Sleep(6 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"available","accountId":"acct-slow","usage":{"total_usage":1.5,"total_credits":10}}`))
	}))
	t.Cleanup(server.Close)

	adapter := NewCursorUsageQuotaAdapter(nil, []string{"127.0.0.1"}, "acct-slow")
	ctx, cancel := context.WithTimeout(context.Background(), cursorAccountQueryTimeout)
	defer cancel()

	snapshot, err := adapter.FetchQuota(ctx, server.URL+"/v1", "gw-key")
	if err != nil {
		t.Fatalf("FetchQuota error: %v", err)
	}
	if snapshot.Status != QuotaStatusAvailable {
		t.Fatalf("status = %s, want available (error=%#v)", snapshot.Status, snapshot.Error)
	}
}

func TestQuotaService_QueryCursorCollectBeyondDefaultQuotaTimeout(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher init error: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "usage/collect") {
			http.NotFound(w, r)
			return
		}
		time.Sleep(6 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"available","usage":{"total_usage":41.68,"total_credits":46.15}}`))
	}))
	t.Cleanup(server.Close)

	sealedGw, _ := cipher.SealManagement("cursor-dev", "gw-key")
	repo := &fakeQuotaRepo{
		records: []Record{
			{
				ID:                      "cursor-dev",
				Name:                    "Cursor",
				Type:                    TypeCursor,
				Endpoint:                strings.TrimRight(server.URL, "/") + "/v1",
				Enabled:                 true,
				HasManagementCredential: true,
			},
		},
		managementCreds: map[string][]byte{"cursor-dev": sealedGw},
	}
	admin := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"127.0.0.1"})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := svc.QueryAllAccounts(ctx)
	if err != nil {
		t.Fatalf("QueryAllAccounts: %v", err)
	}
	if len(res.Snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(res.Snapshots))
	}
	got := res.Snapshots[0]
	if got.Status != QuotaStatusAvailable {
		t.Fatalf("status = %s, want available (error=%#v)", got.Status, got.Error)
	}
	usageMetric, ok := findQuotaMetric(got.Quotas, "total_usage")
	if !ok || usageMetric.Used == nil || *usageMetric.Used != 41.68 {
		t.Fatalf("total_usage metric = %+v", usageMetric)
	}
}
