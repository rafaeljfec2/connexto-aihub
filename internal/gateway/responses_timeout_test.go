package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func TestResponsesHandlerMapsHeaderTimeoutTo504(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-time.After(200 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(upstream.Close)

	transport := &http.Transport{ResponseHeaderTimeout: 40 * time.Millisecond}
	client, err := providers.NewClient(config.ProviderOpenCodeGo{
		BaseURL: upstream.URL,
		APIKey:  providerCredential,
	}, &http.Client{Transport: transport})
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}
	t.Cleanup(transport.CloseIdleConnections)

	logger, _ := newTestLogger()
	handler := gateway.NewResponsesHandler(client, logger, time.Second)
	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want 504; body = %q", response.StatusCode, body)
	}
	if !strings.Contains(body, "execution_timeout") {
		t.Fatalf("body = %q, want execution_timeout", body)
	}
	if strings.Contains(body, providerCredential) {
		t.Fatal("timeout response leaked the provider credential")
	}
}
