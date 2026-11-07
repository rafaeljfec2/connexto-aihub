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
		"total_usage":   41.68,
		"total_credits": 46.15,
		"currency":      "USD",
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
