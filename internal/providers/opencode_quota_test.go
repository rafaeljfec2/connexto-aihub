package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func readOpenCodeFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "testdata", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}

func TestOpenCodeQuotaAdapter_MapsGoUsageAndBudgets(t *testing.T) {
	usageBody := readOpenCodeFixture(t, "opencode_go_usage_sanitized.json")
	budgetBody := readOpenCodeFixture(t, "opencode_current_budgets_sanitized.json")
	statusBody := readOpenCodeFixture(t, "opencode_go_status_sanitized.json")

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer oc_sk_test" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/zen/go/v1/usage":
			_, _ = w.Write(usageBody)
		case "/console/api/policies/current-budgets":
			_, _ = w.Write(budgetBody)
		case "/console/api/go/status":
			_, _ = w.Write(statusBody)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := NewOpenCodeQuotaAdapter(server.Client(), []string{"127.0.0.1", "localhost"})
	snapshot, err := adapter.FetchQuota(context.Background(), server.URL+"/inference/go", "oc_sk_test")
	if err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	if snapshot.Status != QuotaStatusAvailable {
		t.Fatalf("status = %s error=%v", snapshot.Status, snapshot.Error)
	}
	if snapshot.Scope != ScopeSubscription {
		t.Fatalf("scope = %s, want subscription", snapshot.Scope)
	}
	if snapshot.ProviderID != TypeOpenCodeGo {
		t.Fatalf("provider = %s", snapshot.ProviderID)
	}
	if snapshot.WorkspaceName == nil || *snapshot.WorkspaceName != "go-plus" {
		t.Fatalf("workspace_name = %v, want go-plus product label", snapshot.WorkspaceName)
	}

	metrics := map[string]ProviderQuotaMetric{}
	for _, metric := range snapshot.Quotas {
		metrics[metric.Metric] = metric
	}
	rolling, ok := metrics["go_rolling"]
	if !ok || rolling.ConsumedPercent == nil || *rolling.ConsumedPercent != 12 {
		t.Fatalf("go_rolling = %+v", rolling)
	}
	if rolling.ResetsAt == nil || *rolling.ResetsAt != "2026-11-04T18:31:46.000Z" {
		t.Fatalf("go_rolling resets_at = %v", rolling.ResetsAt)
	}
	if rolling.Period == nil || *rolling.Period != "rolling" {
		t.Fatalf("go_rolling period = %v", rolling.Period)
	}
	weekly := metrics["go_weekly"]
	if weekly.ConsumedPercent == nil || *weekly.ConsumedPercent != 36 {
		t.Fatalf("go_weekly percent = %v", weekly.ConsumedPercent)
	}
	monthly := metrics["go_monthly"]
	if monthly.ConsumedPercent == nil || *monthly.ConsumedPercent != 24 {
		t.Fatalf("go_monthly percent = %v", monthly.ConsumedPercent)
	}

	budget, ok := metrics["budget_rule_sanitized_budget_1"]
	if !ok {
		t.Fatalf("missing budget metric, quotas=%v", keysOf(metrics))
	}
	if budget.Used == nil || *budget.Used != 15 || budget.Limit == nil || *budget.Limit != 20 {
		t.Fatalf("budget used/limit = %+v", budget)
	}
	if budget.Remaining != nil {
		t.Fatalf("budget remaining must not be derived: %v", *budget.Remaining)
	}
	if budget.ResetsAt == nil || *budget.ResetsAt != "2026-11-01T00:00:00Z" {
		t.Fatalf("budget resets_at = %v (epoch 1793491200000 → RFC3339 UTC)", budget.ResetsAt)
	}

	wantPaths := map[string]bool{
		"/zen/go/v1/usage":                      true,
		"/console/api/policies/current-budgets": true,
		"/console/api/go/status":                true,
	}
	for _, path := range paths {
		if !wantPaths[path] {
			t.Fatalf("unexpected path %s in %v", path, paths)
		}
	}
}

func TestOpenCodeQuotaAdapter_EmptyBudgetsStillAvailable(t *testing.T) {
	usageBody := readOpenCodeFixture(t, "opencode_go_usage_sanitized.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/zen/go/v1/usage":
			_, _ = w.Write(usageBody)
		case "/console/api/policies/current-budgets":
			_, _ = w.Write([]byte(`[]`))
		case "/console/api/go/status":
			_, _ = w.Write([]byte(`{"product":"go-plus"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := NewOpenCodeQuotaAdapter(server.Client(), []string{"127.0.0.1", "localhost"})
	snapshot, err := adapter.FetchQuota(context.Background(), server.URL, "oc_sk_test")
	if err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	if snapshot.Status != QuotaStatusAvailable {
		t.Fatalf("status = %s", snapshot.Status)
	}
	for _, metric := range snapshot.Quotas {
		if len(metric.Metric) >= 7 && metric.Metric[:7] == "budget_" {
			t.Fatalf("unexpected budget metric %s", metric.Metric)
		}
	}
}

func TestOpenCodeQuotaAdapter_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	adapter := NewOpenCodeQuotaAdapter(server.Client(), []string{"127.0.0.1", "localhost"})
	snapshot, err := adapter.FetchQuota(context.Background(), server.URL, "bad")
	if err != nil {
		t.Fatalf("FetchQuota error: %v", err)
	}
	if snapshot.Status != QuotaStatusUnauthorized {
		t.Fatalf("status = %s", snapshot.Status)
	}
}

func TestOpenCodeQuotaAdapter_HostNotAllowed(t *testing.T) {
	adapter := NewOpenCodeQuotaAdapter(http.DefaultClient, []string{"opencode.ai"})
	snapshot, err := adapter.FetchQuota(context.Background(), "https://evil.example/inference/go", "oc_sk_test")
	if err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	if snapshot.Status != QuotaStatusError || snapshot.Error == nil || snapshot.Error.Code != "host_not_allowed" {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestOpenCodeQuotaAdapter_MissingCredential(t *testing.T) {
	adapter := NewOpenCodeQuotaAdapter(nil, []string{"opencode.ai"})
	snapshot, err := adapter.FetchQuota(context.Background(), "https://opencode.ai/inference/go", "")
	if err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	if snapshot.Status != QuotaStatusUnconfiguredCredential {
		t.Fatalf("status = %s", snapshot.Status)
	}
}

func TestOpenCodeQuotaBaseURLStripsInferenceSuffix(t *testing.T) {
	if got := openCodeQuotaRoot("https://opencode.ai/inference/go"); got != "https://opencode.ai" {
		t.Fatalf("got %s", got)
	}
	if got := openCodeQuotaRoot("https://opencode.ai/inference/go/"); got != "https://opencode.ai" {
		t.Fatalf("got %s", got)
	}
	if got := openCodeQuotaRoot(""); got != "https://opencode.ai" {
		t.Fatalf("got %s", got)
	}
}

func TestQuotaService_OpenCodeReturnsSingleSubscriptionSnapshot(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	sealed, err := cipher.Seal("opencode-go", "oc_sk_live")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	repo := &fakeQuotaRepo{
		records: []Record{{
			ID: OpenCodeGoID, Name: "OpenCode Go", Type: TypeOpenCodeGo,
			Endpoint: "https://opencode.ai/inference/go", Enabled: true, HasCredential: true,
		}},
		creds: map[string][]byte{OpenCodeGoID: sealed},
	}
	admin := NewService(repo, cipher, NewRegistry(), nil, []string{"opencode.ai"})
	adapter := &fakeMockAdapter{fn: func(_ context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
		if endpoint != "https://opencode.ai/inference/go" {
			t.Fatalf("endpoint = %s", endpoint)
		}
		if credential != "oc_sk_live" {
			t.Fatalf("credential mismatch")
		}
		return ProviderQuotaSnapshot{
			Status: QuotaStatusAvailable, Scope: ScopeSubscription,
			Quotas: []ProviderQuotaMetric{{Metric: "go_rolling", Label: "Go rolling", Nature: NatureQuota}},
			Source: ProviderQuotaSource{Type: "provider_api", Name: "opencode_go_usage_api", Scope: ScopeSubscription},
		}, nil
	}}
	service := NewQuotaService(admin, map[string]QuotaAdapter{TypeOpenCodeGo: adapter}, nil)
	response, err := service.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("QueryAllAccounts: %v", err)
	}
	if len(response.Snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1 (subscription only)", len(response.Snapshots))
	}
	if response.Snapshots[0].Scope != ScopeSubscription {
		t.Fatalf("scope = %s", response.Snapshots[0].Scope)
	}
	if response.Snapshots[0].Source.Name != "opencode_go_usage_api" {
		t.Fatalf("source = %s", response.Snapshots[0].Source.Name)
	}
}

func TestLiveOpenCodeGoQuotaSmoke(t *testing.T) {
	if os.Getenv("AIHUB_LIVE_SMOKE") != "1" {
		t.Skip("skipped: set AIHUB_LIVE_SMOKE=1 for live OpenCode quota smoke")
	}
	credential := os.Getenv("AIHUB_PROVIDER_OPENCODEGO_API_KEY")
	if credential == "" {
		credential = os.Getenv("OPENCODE_API_KEY")
	}
	if credential == "" {
		t.Skip("skipped: no OpenCode API key")
	}
	adapter := NewOpenCodeQuotaAdapter(&http.Client{Timeout: 10 * time.Second}, []string{"opencode.ai"})
	snapshot, err := adapter.FetchQuota(context.Background(), "https://opencode.ai/inference/go", credential)
	if err != nil {
		t.Fatalf("FetchQuota: %v", err)
	}
	if snapshot.Status != QuotaStatusAvailable {
		t.Fatalf("status = %s error=%v", snapshot.Status, snapshot.Error)
	}
	found := false
	for _, metric := range snapshot.Quotas {
		if metric.Metric == "go_rolling" && metric.ConsumedPercent != nil {
			found = true
		}
	}
	if !found {
		t.Fatalf("live snapshot missing go_rolling percent: %+v", snapshot.Quotas)
	}
}

func keysOf(metrics map[string]ProviderQuotaMetric) []string {
	out := make([]string, 0, len(metrics))
	for key := range metrics {
		out = append(out, key)
	}
	return out
}
