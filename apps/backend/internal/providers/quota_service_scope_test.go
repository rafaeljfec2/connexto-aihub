package providers

import (
	"context"
	"sync"
	"testing"
)

type scopedQuotaTestAdapter struct {
	mu               sync.Mutex
	keyCredentials   map[string]string
	adminCredentials map[string]string
}

func (a *scopedQuotaTestAdapter) FetchQuota(_ context.Context, _ string, credential string) (ProviderQuotaSnapshot, error) {
	a.mu.Lock()
	a.keyCredentials[credential] = credential
	a.mu.Unlock()
	if credential == "key-b" {
		return ProviderQuotaSnapshot{Scope: ScopeAPIKey, Status: QuotaStatusUnauthorized}, nil
	}
	usage := 3.5
	return ProviderQuotaSnapshot{Scope: ScopeAPIKey, Status: QuotaStatusAvailable,
		Quotas: []ProviderQuotaMetric{{Metric: "usage", Used: &usage}},
		Source: ProviderQuotaSource{Scope: ScopeAPIKey, Name: "openrouter_key_api"}}, nil
}

func (a *scopedQuotaTestAdapter) FetchAccountQuota(_ context.Context, _ string, credential string) (ProviderQuotaSnapshot, error) {
	a.mu.Lock()
	a.adminCredentials[credential] = credential
	a.mu.Unlock()
	credits := 10.0
	if credential == "admin-b" {
		credits = 20
	}
	return ProviderQuotaSnapshot{Scope: ScopeAccount, Status: QuotaStatusAvailable,
		Quotas: []ProviderQuotaMetric{{Metric: "total_credits", Limit: &credits}},
		Source: ProviderQuotaSource{Scope: ScopeAccount, Name: "openrouter_credits_api"}}, nil
}

func TestQuotaServiceKeepsAccountAndKeyScopesIndependentAcrossAccounts(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}
	keyA, _ := cipher.Seal("openrouter-a", "key-a")
	keyB, _ := cipher.Seal("openrouter-b", "key-b")
	adminA, _ := cipher.SealManagement("openrouter-a", "admin-a")
	adminB, _ := cipher.SealManagement("openrouter-b", "admin-b")
	repo := &fakeQuotaRepo{
		records: []Record{
			{ID: "openrouter-a", Name: "Account A", Type: TypeOpenRouter, Endpoint: "https://openrouter.ai/api/v1", Enabled: true, HasCredential: true, HasManagementCredential: true},
			{ID: "openrouter-b", Name: "Account B", Type: TypeOpenRouter, Endpoint: "https://openrouter.ai/api/v1", Enabled: true, HasCredential: true, HasManagementCredential: true},
		},
		creds:           map[string][]byte{"openrouter-a": keyA, "openrouter-b": keyB},
		managementCreds: map[string][]byte{"openrouter-a": adminA, "openrouter-b": adminB},
	}
	adminService := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"openrouter.ai"})
	adapter := &scopedQuotaTestAdapter{keyCredentials: map[string]string{}, adminCredentials: map[string]string{}}
	service := NewQuotaService(adminService, map[string]QuotaAdapter{TypeOpenRouter: adapter}, nil)

	response, err := service.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("QueryAllAccounts: %v", err)
	}
	if len(response.Snapshots) != 4 {
		t.Fatalf("snapshot count = %d, want 4", len(response.Snapshots))
	}
	byAccountScope := make(map[string]ProviderQuotaSnapshot, len(response.Snapshots))
	for _, snapshot := range response.Snapshots {
		byAccountScope[snapshot.AccountID+":"+string(snapshot.Scope)] = snapshot
	}
	if byAccountScope["openrouter-a:api_key"].Quotas[0].Used == nil || *byAccountScope["openrouter-a:api_key"].Quotas[0].Used != 3.5 {
		t.Fatal("account A key usage was not kept in its api_key snapshot")
	}
	if byAccountScope["openrouter-b:api_key"].Status != QuotaStatusUnauthorized {
		t.Fatal("account B key failure was not isolated to its api_key snapshot")
	}
	if byAccountScope["openrouter-b:account"].Quotas[0].Limit == nil || *byAccountScope["openrouter-b:account"].Quotas[0].Limit != 20 {
		t.Fatal("account B credits were mixed with another account or key scope")
	}
	if len(adapter.keyCredentials) != 2 || adapter.keyCredentials["key-a"] == "" || adapter.keyCredentials["key-b"] == "" {
		t.Fatal("inference credentials were not independently used by /key")
	}
	if len(adapter.adminCredentials) != 2 || adapter.adminCredentials["admin-a"] == "" || adapter.adminCredentials["admin-b"] == "" {
		t.Fatal("management credentials were not independently used by /credits")
	}
}
