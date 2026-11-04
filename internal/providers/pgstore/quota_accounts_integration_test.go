package pgstore_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/providers"
	"aihub/internal/providers/pgstore"
)

func TestQuotaReadsPersistAndSeparateMultipleOpenRouterAccountsIntegration(t *testing.T) {
	pool := migratedPool(t)
	cipher, err := providers.NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/api/v1/key":
			if strings.HasPrefix(authorization, "Bearer key-a") {
				_, _ = w.Write([]byte(`{"data":{"usage":1.25}}`))
				return
			}
			if strings.HasPrefix(authorization, "Bearer key-b") {
				_, _ = w.Write([]byte(`{"data":{"usage":2.5}}`))
				return
			}
		case "/api/v1/credits":
			if strings.HasPrefix(authorization, "Bearer admin-a") {
				_, _ = w.Write([]byte(`{"data":{"total_credits":10,"total_usage":4.25}}`))
				return
			}
			if strings.HasPrefix(authorization, "Bearer admin-b") {
				_, _ = w.Write([]byte(`{"data":{"total_credits":20,"total_usage":8.75}}`))
				return
			}
		}
		t.Errorf("unexpected route or credential scope for %s", r.URL.Path)
		http.Error(w, "unauthorized", http.StatusForbidden)
	}))
	defer server.Close()

	admin := providers.NewServiceWithProbes(pgstore.New(pool), cipher, providers.NewRegistry(), nil,
		[]string{"127.0.0.1", "localhost"})
	for _, account := range []struct {
		id, name, key, management string
	}{
		{"openrouter-a", "OpenRouter A", "key-a-inference", "admin-a-management"},
		{"openrouter-b", "OpenRouter B", "key-b-inference", "admin-b-management"},
	} {
		if _, err := admin.Create(context.Background(), providers.CreateInput{
			ID: account.id, Name: account.name, Type: providers.TypeOpenRouter,
			Endpoint: server.URL + "/api/v1", Credential: account.key,
			ManagementCredential: account.management, Enabled: true,
		}); err != nil {
			t.Fatalf("create %s: %v", account.id, err)
		}
	}

	quota := providers.NewQuotaService(admin, map[string]providers.QuotaAdapter{
		providers.TypeOpenRouter: providers.NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1"}),
	}, nil)
	response, err := quota.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("QueryAllAccounts: %v", err)
	}
	if len(response.Snapshots) != 4 {
		t.Fatalf("snapshot count = %d, want four independent account/scope results", len(response.Snapshots))
	}
	readings := make(map[string]providers.ProviderQuotaSnapshot, len(response.Snapshots))
	for _, snapshot := range response.Snapshots {
		readings[snapshot.AccountID+":"+string(snapshot.Scope)] = snapshot
	}
	accountAKeyUsage, ok := findQuotaMetric(readings["openrouter-a:api_key"], "usage")
	if !ok || accountAKeyUsage.Used == nil {
		t.Fatalf("account A key usage metric = %+v, want usage value", accountAKeyUsage)
	}
	if got := *accountAKeyUsage.Used; got != 1.25 {
		t.Fatalf("account A key usage = %v, want 1.25", got)
	}
	accountACredits, ok := findQuotaMetric(readings["openrouter-a:account"], "total_credits")
	if !ok || accountACredits.Limit == nil || *accountACredits.Limit != 10 {
		t.Fatalf("account A credits metric = %+v, want total_credits limit 10", accountACredits)
	}
	accountBKeyUsage, ok := findQuotaMetric(readings["openrouter-b:api_key"], "usage")
	if !ok || accountBKeyUsage.Used == nil {
		t.Fatalf("account B key usage metric = %+v, want usage value", accountBKeyUsage)
	}
	if got := *accountBKeyUsage.Used; got != 2.5 {
		t.Fatalf("account B key usage = %v, want 2.5", got)
	}
	accountBUsage, ok := findQuotaMetric(readings["openrouter-b:account"], "total_usage")
	if !ok || accountBUsage.Used == nil {
		t.Fatalf("account B account usage metric = %+v, want total_usage value", accountBUsage)
	}
	if got := *accountBUsage.Used; got != 8.75 {
		t.Fatalf("account B account usage = %v, want 8.75", got)
	}

	assertStoredManagementCredential(t, pool, cipher, "openrouter-a", "key-a-inference", "admin-a-management")
	assertStoredManagementCredential(t, pool, cipher, "openrouter-b", "key-b-inference", "admin-b-management")
}

func findQuotaMetric(snapshot providers.ProviderQuotaSnapshot, name string) (providers.ProviderQuotaMetric, bool) {
	for _, metric := range snapshot.Quotas {
		if metric.Metric == name {
			return metric, true
		}
	}
	return providers.ProviderQuotaMetric{}, false
}

func assertStoredManagementCredential(t *testing.T, pool *pgxpool.Pool, cipher *providers.CredentialCipher,
	id, inference, management string) {
	t.Helper()
	var inferenceCiphertext, managementCiphertext []byte
	if err := pool.QueryRow(context.Background(), `SELECT credential_ciphertext, management_credential_ciphertext FROM providers WHERE id = $1`, id).
		Scan(&inferenceCiphertext, &managementCiphertext); err != nil {
		t.Fatalf("read protected credentials for %s: %v", id, err)
	}
	if strings.Contains(string(inferenceCiphertext), inference) || strings.Contains(string(managementCiphertext), management) {
		t.Fatal("provider credentials are stored in plaintext")
	}
	if opened, err := cipher.Open(id, inferenceCiphertext); err != nil || opened != inference {
		t.Fatalf("inference credential did not survive persistence: %v", err)
	}
	if opened, err := cipher.OpenManagement(id, managementCiphertext); err != nil || opened != management {
		t.Fatalf("management credential did not survive persistence: %v", err)
	}
	if _, err := cipher.Open(id, managementCiphertext); err == nil {
		t.Fatal("management credential ciphertext opened in the inference context")
	}
}
