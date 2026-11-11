package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenRouterQuotaAdapter_Success(t *testing.T) {
	fixture := `{
		"data": {
			"label": "sk-or-v1-my-team-key",
			"usage": 25.5,
			"limit": 100.0,
			"limit_remaining": 74.5,
			"limit_reset": "monthly",
			"usage_daily": 2.1,
			"usage_weekly": 15.0,
			"usage_monthly": 25.5,
			"byok_usage": 0.5,
			"byok_usage_daily": 0.1,
			"byok_usage_weekly": 0.2,
			"byok_usage_monthly": 0.5,
			"is_free_tier": false
		}
	}`

	requestedPaths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/api/v1/key" {
			_, _ = w.Write([]byte(fixture))
			return
		}
		if r.URL.Path == "/api/v1/credits" {
			_, _ = w.Write([]byte(`{"data":{"total_credits":10.0,"total_usage":4.84}}`))
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer server.Close()

	// Parse server host to add to allowlist
	adapter := NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1", "localhost"})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	snapshot, err := adapter.FetchQuota(ctx, server.URL+"/api/v1", "test-api-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Status != QuotaStatusAvailable {
		t.Fatalf("expected status available, got %s", snapshot.Status)
	}
	if snapshot.Scope != ScopeAPIKey {
		t.Errorf("expected scope api_key, got %s", snapshot.Scope)
	}
	if snapshot.ProviderID != TypeOpenRouter {
		t.Errorf("expected provider openrouter, got %s", snapshot.ProviderID)
	}
	if len(requestedPaths) != 1 || requestedPaths[0] != "/api/v1/key" {
		t.Fatalf("key-scoped read requested %v, want only /api/v1/key", requestedPaths)
	}

	// Verify that NO percent calculations were performed
	for _, q := range snapshot.Quotas {
		if q.ConsumedPercent != nil {
			t.Errorf("metric %s has non-nil ConsumedPercent (%v), calculations are forbidden", q.Metric, *q.ConsumedPercent)
		}
		if q.RemainingPercent != nil {
			t.Errorf("metric %s has non-nil RemainingPercent (%v), calculations are forbidden", q.Metric, *q.RemainingPercent)
		}
		if q.Unit == nil || *q.Unit != "USD" {
			t.Errorf("metric %s expected unit USD, got %v", q.Metric, q.Unit)
		}
	}

	// The key-scoped response must not include account-scoped credits.
	if len(snapshot.Quotas) != 10 {
		t.Fatalf("expected 10 key-scoped metrics, got %d", len(snapshot.Quotas))
	}

	metricsMap := make(map[string]ProviderQuotaMetric)
	for _, m := range snapshot.Quotas {
		metricsMap[m.Metric] = m
	}

	limitMetric, ok := metricsMap["limit"]
	if !ok || limitMetric.Limit == nil || *limitMetric.Limit != 100.0 {
		t.Errorf("limit metric mismatch: %+v", limitMetric)
	}
	if limitMetric.Period == nil || *limitMetric.Period != "monthly" {
		t.Errorf("limit reset period mismatch: %+v", limitMetric.Period)
	}

	remMetric, ok := metricsMap["limit_remaining"]
	if !ok || remMetric.Remaining == nil || *remMetric.Remaining != 74.5 {
		t.Errorf("limit_remaining metric mismatch: %+v", remMetric)
	}

	usageMetric, ok := metricsMap["usage"]
	if !ok || usageMetric.Used == nil || *usageMetric.Used != 25.5 {
		t.Errorf("usage metric mismatch: %+v", usageMetric)
	}

	dailyMetric, ok := metricsMap["usage_daily"]
	if !ok || dailyMetric.Used == nil || *dailyMetric.Used != 2.1 {
		t.Errorf("usage_daily metric mismatch: %+v", dailyMetric)
	}
	if _, found := metricsMap["total_credits"]; found {
		t.Fatal("account-scoped credits were mixed into the api_key snapshot")
	}
}

func TestOpenRouterQuotaAdapter_PreservesZeroAndOmittedFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"usage":0}}`))
	}))
	defer server.Close()

	adapter := NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1"})
	snapshot, err := adapter.FetchQuota(context.Background(), server.URL, "test-key")
	if err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}

	metrics := make(map[string]ProviderQuotaMetric, len(snapshot.Quotas))
	for _, metric := range snapshot.Quotas {
		metrics[metric.Metric] = metric
	}
	usage, found := metrics["usage"]
	if !found || usage.Used == nil || *usage.Used != 0 {
		t.Fatalf("explicit zero usage was not preserved: %+v", usage)
	}
	if _, found := metrics["usage_daily"]; found {
		t.Fatal("omitted usage_daily field was converted into a metric")
	}
}

func TestOpenRouterQuotaAdapter_HostValidationRejection(t *testing.T) {
	adapter := NewOpenRouterQuotaAdapter(nil, []string{"openrouter.ai"})

	snapshot, err := adapter.FetchQuota(context.Background(), "https://malicious-host.com/api/v1", "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Status != QuotaStatusError {
		t.Fatalf("expected status error, got %s", snapshot.Status)
	}
	if snapshot.Error == nil || snapshot.Error.Code != "host_not_allowed" {
		t.Fatalf("expected host_not_allowed error code, got %+v", snapshot.Error)
	}
}

func TestOpenRouterQuotaAdapter_UnconfiguredCredential(t *testing.T) {
	adapter := NewOpenRouterQuotaAdapter(nil, []string{"openrouter.ai"})

	snapshot, err := adapter.FetchQuota(context.Background(), "https://openrouter.ai/api/v1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Status != QuotaStatusUnconfiguredCredential {
		t.Fatalf("expected unconfigured_credential, got %s", snapshot.Status)
	}
}

func TestOpenRouterQuotaAdapter_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer server.Close()

	adapter := NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1", "localhost"})

	snapshot, err := adapter.FetchQuota(context.Background(), server.URL, "bad-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Status != QuotaStatusUnauthorized {
		t.Fatalf("expected unauthorized, got %s", snapshot.Status)
	}
}

func TestOpenRouterQuotaAdapter_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Rate limit exceeded"}}`))
	}))
	defer server.Close()

	adapter := NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1", "localhost"})

	snapshot, err := adapter.FetchQuota(context.Background(), server.URL, "my-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Status != QuotaStatusRateLimited {
		t.Fatalf("expected rate_limited, got %s", snapshot.Status)
	}
}

func TestOpenRouterQuotaAdapter_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1", "localhost"})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	snapshot, err := adapter.FetchQuota(ctx, server.URL, "my-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Status != QuotaStatusTimeout {
		t.Fatalf("expected timeout, got %s", snapshot.Status)
	}
}
