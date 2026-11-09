package gateway_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func newUsageAnalyticsAdminServer(t *testing.T) (*httptest.Server, *providers.CursorUsageAnalyticsService) {
	t.Helper()
	cipher, err := providers.NewCredentialCipher(bytes.Repeat([]byte{0x22}, config.CredentialKeyBytes))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}
	repo := &inMemoryProviders{
		records: map[string]providers.Record{}, sealed: map[string][]byte{},
		sealedManagement: map[string][]byte{},
	}
	service := providers.NewService(repo, cipher, providers.NewRegistry(), providers.ProbeEndpoint,
		[]string{"127.0.0.1"})
	analytics := providers.NewCursorUsageAnalyticsService(service, http.DefaultClient, nil)
	admin := gateway.NewProviderAdmin(service, newAdminLogger())
	admin.BindCursorUsageAnalytics(analytics)
	mux := http.NewServeMux()
	mux.Handle("GET /api/ui/providers/cursor/usage-analytics", admin.CursorUsageAnalytics())
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, analytics
}

func TestCursorUsageAnalyticsRoute_MethodGuard(t *testing.T) {
	server, _ := newUsageAnalyticsAdminServer(t)

	response, err := http.Post(server.URL+"/api/ui/providers/cursor/usage-analytics", "application/json", nil)
	if err != nil {
		t.Fatalf("POST usage analytics: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST, got %d", response.StatusCode)
	}
}

func TestCursorUsageAnalyticsRoute_ReturnsEnvelopeWithoutConfiguredProviders(t *testing.T) {
	server, _ := newUsageAnalyticsAdminServer(t)

	response, err := http.Get(server.URL + "/api/ui/providers/cursor/usage-analytics")
	if err != nil {
		t.Fatalf("GET usage analytics: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

	var envelope providers.CursorUsageAnalyticsEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Status != providers.UsageAnalyticsStatusProviderNotFound {
		t.Fatalf("expected provider_not_found, got %q", envelope.Status)
	}
	if envelope.FetchedAt == "" {
		t.Fatal("expected envelope to carry a fetchedAt timestamp")
	}
}

func TestCursorUsageAnalyticsRoute_UnknownExplicitProviderIsNotFound(t *testing.T) {
	server, _ := newUsageAnalyticsAdminServer(t)

	response, err := http.Get(server.URL + "/api/ui/providers/cursor/usage-analytics?refresh=true&providerId=missing")
	if err != nil {
		t.Fatalf("GET usage analytics: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown provider id, got %d", response.StatusCode)
	}
}
