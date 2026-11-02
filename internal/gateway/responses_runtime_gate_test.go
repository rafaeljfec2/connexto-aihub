package gateway_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

// A configuration disabled at runtime refuses new executions, while the same
// client accepted them before the change (Phase C1 runtime gate).
func TestResponsesHandlerRefusesNewExecutionsAfterRuntimeDisable(t *testing.T) {
	logger, _ := newTestLogger()

	client, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: "https://example.com/go", APIKey: providerCredential}, nil)
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}

	if err := client.Reconfigure(providers.Settings{BaseURL: "https://example.com/go", APIKey: providerCredential, Enabled: false}); err != nil {
		t.Fatalf("Reconfigure() returned error: %v", err)
	}

	handler := gateway.NewResponsesHandler(client, logger, time.Minute)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 for a disabled provider", response.StatusCode)
	}

	if !strings.Contains(body, "provider_unavailable") || strings.Contains(body, providerCredential) {
		t.Fatalf("body %q must mention provider_unavailable without the credential", body)
	}
}
