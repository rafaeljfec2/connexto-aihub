package gateway_test

import (
	"bytes"
	"context"
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
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamHits++
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	endpoint := upstream.URL + "/api/v1"

	repo := &inMemoryProviders{records: map[string]providers.Record{}, sealed: map[string][]byte{}}
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
		providers.TypeOpenRouter: func(context.Context, string, string) error {
			probeCalls[providers.TypeOpenRouter]++
			return nil
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
	server := httptest.NewServer(mux)
	defer server.Close()

	status, body := adminCall(t, http.MethodPost, server.URL+"/api/providers/openrouter/validate", "")
	if status != http.StatusOK || !strings.Contains(body, "reachable") {
		t.Fatalf("OpenRouter validation = %d %s, want sanitized success", status, body)
	}
	if probeCalls[providers.TypeOpenCodeGo] != 0 || probeCalls[providers.TypeOpenRouter] != 1 {
		t.Fatalf("validation probes = OpenCode %d, OpenRouter %d", probeCalls[providers.TypeOpenCodeGo], probeCalls[providers.TypeOpenRouter])
	}
	assertGatewayProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnknown)
	assertGatewayProviderState(t, manager, providers.OpenRouterID, providers.StateHealthy)

	status, body = adminCall(t, http.MethodPost, server.URL+"/api/providers/opencode-go/validate", "")
	if status != http.StatusBadGateway || strings.Contains(body, adminSecret) || strings.Contains(logBuffer.String(), adminSecret) {
		t.Fatalf("failed validation exposed sensitive data: status=%d body=%q logs=%q", status, body, logBuffer.String())
	}
	assertGatewayProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnhealthy)
	assertGatewayProviderState(t, manager, providers.OpenRouterID, providers.StateHealthy)
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
