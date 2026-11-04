package gateway_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func TestProviderValidationUpdatesOnlyTargetStateAndSanitizesErrors(t *testing.T) {
	upstreamHits := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits++
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/models" {
			t.Errorf("probe request = %s %s, want GET /api/v1/models", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+adminSecret {
			t.Error("OpenRouter probe authorization header mismatch")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	endpoint := upstream.URL + "/api/v1"

	repo := &inMemoryProviders{
		records:          map[string]providers.Record{},
		sealed:           map[string][]byte{},
		sealedManagement: map[string][]byte{},
	}
	registry := providers.NewRegistry()
	openCode, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: endpoint, APIKey: adminSecret}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	openRouter, err := providers.NewOpenRouterClient(providers.Settings{BaseURL: endpoint, APIKey: adminSecret, Enabled: true}, nil)
	if err != nil {
		t.Fatalf("NewOpenRouterClient: %v", err)
	}
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.OpenRouterID, openRouter)
	manager := providers.NewRegistryManager(nil, registry)
	probeCalls := map[string]int{}
	service := providers.NewServiceWithProbes(repo, adminCipher(t), registry, map[string]providers.ProbeFunc{
		providers.TypeOpenCodeGo: func(context.Context, string, string) error {
			probeCalls[providers.TypeOpenCodeGo]++
			return errors.New(adminSecret)
		},
		providers.TypeOpenRouter: func(ctx context.Context, endpoint, credential string) error {
			probeCalls[providers.TypeOpenRouter]++
			return providers.ProbeOpenRouterEndpoint(ctx, endpoint, credential)
		},
	}, []string{"127.0.0.1"})
	for _, input := range []providers.BootstrapInput{
		{ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo, Endpoint: endpoint, Credential: adminSecret, Enabled: true},
		{ID: providers.OpenRouterID, Name: "OpenRouter", Type: providers.TypeOpenRouter, Endpoint: endpoint, Credential: adminSecret, Enabled: true},
	} {
		if _, err := service.Bootstrap(context.Background(), input); err != nil {
			t.Fatalf("Bootstrap(%s): %v", input.ID, err)
		}
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Manager.Start: %v", err)
	}
	if upstreamHits != 0 {
		t.Fatalf("startup made %d upstream requests, want zero", upstreamHits)
	}
	assertGatewayProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnknown)
	assertGatewayProviderState(t, manager, providers.OpenRouterID, providers.StateUnknown)

	logBuffer := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(logBuffer, nil))
	admin := gateway.NewProviderAdmin(service, logger)
	mux := http.NewServeMux()
	mux.Handle("POST /api/providers/{id}/validate", admin.Validate())
	mux.Handle("GET /diagnostics/providers", gateway.NewProvidersHandler(manager.Statuses))
	server := httptest.NewServer(mux)
	defer server.Close()

	assertProviderDiagnosticsState(t, server.URL, providers.OpenRouterID, true, providers.StateUnknown)
	status, body := adminCall(t, http.MethodPost, server.URL+"/api/providers/openrouter/validate", "")
	if status != http.StatusOK || !strings.Contains(body, "reachable") {
		t.Fatalf("OpenRouter validation = %d %s, want a reachable result", status, body)
	}
	if probeCalls[providers.TypeOpenCodeGo] != 0 || probeCalls[providers.TypeOpenRouter] != 1 {
		t.Fatalf("validation probes = OpenCode %d, OpenRouter %d", probeCalls[providers.TypeOpenCodeGo], probeCalls[providers.TypeOpenRouter])
	}
	if upstreamHits != 1 {
		t.Fatalf("OpenRouter validation made %d upstream requests, want 1", upstreamHits)
	}
	assertGatewayProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnknown)
	assertGatewayProviderState(t, manager, providers.OpenRouterID, providers.StateHealthy)
	assertProviderDiagnosticsState(t, server.URL, providers.OpenRouterID, true, providers.StateHealthy)

	status, body = adminCall(t, http.MethodPost, server.URL+"/api/providers/opencode-go/validate", "")
	if status != http.StatusBadGateway || strings.Contains(body, adminSecret) || strings.Contains(logBuffer.String(), adminSecret) {
		t.Fatalf("failed validation exposed sensitive data: status=%d body=%q logs=%q", status, body, logBuffer.String())
	}
	assertGatewayProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnhealthy)
	assertGatewayProviderState(t, manager, providers.OpenRouterID, providers.StateHealthy)
	if upstreamHits != 1 {
		t.Fatalf("OpenCode Go validation made an unexpected upstream request; total hits = %d", upstreamHits)
	}
}

func assertProviderDiagnosticsState(t *testing.T, serverURL, providerID string, wantEnabled bool, wantState providers.State) {
	t.Helper()

	status, body := adminCall(t, http.MethodGet, serverURL+"/diagnostics/providers", "")
	if status != http.StatusOK {
		t.Fatalf("provider diagnostics status = %d, want %d: %s", status, http.StatusOK, body)
	}
	var response struct {
		Providers []providers.Status `json:"providers"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode provider diagnostics: %v", err)
	}
	for _, provider := range response.Providers {
		if provider.ID != providerID {
			continue
		}
		if provider.Enabled != wantEnabled || provider.State != wantState {
			t.Fatalf("provider diagnostics = %+v, want enabled=%t state=%q", provider, wantEnabled, wantState)
		}
		return
	}
	t.Fatalf("provider %q missing from diagnostics: %+v", providerID, response.Providers)
}

func adminCipher(t *testing.T) *providers.CredentialCipher {
	t.Helper()
	cipher, err := providers.NewCredentialCipher(bytes.Repeat([]byte{0x23}, config.CredentialKeyBytes))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}
	return cipher
}

func assertGatewayProviderState(t *testing.T, manager *providers.Manager, providerID string, want providers.State) {
	t.Helper()
	for _, status := range manager.Statuses() {
		if status.ID == providerID {
			if status.State != want {
				t.Fatalf("provider %s state = %q, want %q", providerID, status.State, want)
			}
			return
		}
	}
	t.Fatalf("provider %s missing from manager statuses", providerID)
}
