package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type openRouterAccountQuotaReader interface {
	FetchAccountQuota(context.Context, string, string) (ProviderQuotaSnapshot, error)
}

func TestOpenRouterAccountQuotaDoesNotReturnProviderErrorBody(t *testing.T) {
	const secretMarker = "management-secret-must-not-return"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"` + secretMarker + `"}`))
	}))
	defer server.Close()

	adapter := NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1"})
	reader, ok := interface{}(adapter).(openRouterAccountQuotaReader)
	if !ok {
		t.Fatal("OpenRouter quota adapter does not expose an independent account-scoped read")
	}
	snapshot, err := reader.FetchAccountQuota(context.Background(), server.URL, "management-key")
	if err != nil {
		t.Fatalf("FetchAccountQuota: %v", err)
	}
	if snapshot.Status != QuotaStatusUnauthorized || snapshot.Error == nil || strings.Contains(snapshot.Error.Message, secretMarker) {
		t.Fatalf("provider error body was returned: %+v", snapshot.Error)
	}
}

func TestOpenRouterAccountQuotaUsesManagementCredentialAndAccountScope(t *testing.T) {
	const managementCredential = "management-key-test-only"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/credits" {
			t.Errorf("path = %q, want account credits endpoint", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+managementCredential {
			t.Errorf("authorization header did not use the Management API Key")
		}
		_, _ = w.Write([]byte(`{"data":{"total_credits":10,"total_usage":4.84186443}}`))
	}))
	defer server.Close()

	adapter := NewOpenRouterQuotaAdapter(server.Client(), []string{"127.0.0.1"})
	reader, ok := interface{}(adapter).(openRouterAccountQuotaReader)
	if !ok {
		t.Fatal("OpenRouter quota adapter does not expose an independent account-scoped read")
	}

	snapshot, err := reader.FetchAccountQuota(context.Background(), server.URL+"/api/v1", managementCredential)
	if err != nil {
		t.Fatalf("FetchAccountQuota: %v", err)
	}
	if snapshot.Scope != ScopeAccount || snapshot.Source.Scope != ScopeAccount {
		t.Fatalf("account credits scope = %q / %q, want account", snapshot.Scope, snapshot.Source.Scope)
	}
	if len(snapshot.Quotas) != 2 || snapshot.Quotas[1].Used == nil || *snapshot.Quotas[1].Used != 4.84186443 {
		t.Fatalf("account credits values were not preserved: %+v", snapshot.Quotas)
	}
}
