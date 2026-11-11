package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenRouterQuotaAdapter_PreservesBudgetScopesTimestampsAndRequestMeter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{
			"usage":0,
			"budgets":[{"limit_usd":50,"spend_usd":32.5,"remaining_usd":17.5,"reset_interval":"weekly","resets_at":"2026-11-10T00:00:00.000Z","scope":"member","include_byok":false}],
			"free_model_daily_requests":{"used":0,"limit":50,"remaining":38},
			"expires_at":"2027-12-31T23:59:59Z"
		}}`))
	}))
	defer server.Close()

	adapter := NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1"})
	snapshot, err := adapter.FetchQuota(context.Background(), server.URL+"/api/v1", "inference-key")
	if err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	metrics := make(map[string]ProviderQuotaMetric, len(snapshot.Quotas))
	for _, metric := range snapshot.Quotas {
		metrics[metric.Metric] = metric
	}
	budget := metrics["budget_1"]
	if budget.Scope != "member" || budget.Used == nil || *budget.Used != 32.5 || budget.ResetsAt == nil || *budget.ResetsAt != "2026-11-10T00:00:00.000Z" {
		t.Fatalf("budget fields were not preserved: %+v", budget)
	}
	if budget.IncludeBYOK == nil || *budget.IncludeBYOK {
		t.Fatalf("explicit false include_byok was not preserved: %+v", budget)
	}
	requests := metrics["free_model_daily_requests"]
	if requests.Used == nil || *requests.Used != 0 || requests.Unit == nil || *requests.Unit != "requests" {
		t.Fatalf("request meter did not preserve its explicit zero and unit: %+v", requests)
	}
	if snapshot.ProviderExpiresAt == nil || *snapshot.ProviderExpiresAt != "2027-12-31T23:59:59Z" {
		t.Fatalf("provider expiration timestamp was not preserved: %+v", snapshot.ProviderExpiresAt)
	}
}
