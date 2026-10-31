package providers_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/providers"
)

func TestOpenTargetResponsesPreservesPayloadAndTargetDoesNotRewriteIt(t *testing.T) {
	const payload = " { \"model\" : \"incoming-model\", \"stream\" : true }\n"
	var received string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(upstream.Close)
	client := mustClient(t, newEnabledClientConfig(upstream.URL))

	response, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenCodeGoID,
		ModelID:    "resolved-model",
	}, []byte(payload), "application/json", "session-target-test")
	if err != nil {
		t.Fatalf("OpenTargetResponses() returned error: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", response.StatusCode)
	}
	if received != payload {
		t.Fatalf("payload = %q, want byte-identical %q", received, payload)
	}
}

func TestOpenTargetResponsesRejectsInvalidTargetIdentity(t *testing.T) {
	client := mustClient(t, newEnabledClientConfig("http://127.0.0.1:1"))
	tests := []providers.ResponsesTarget{
		{ProviderID: "unknown"},
		{ProviderID: providers.OpenCodeGoID, ModelID: " \t "},
	}
	for _, target := range tests {
		if _, err := client.OpenTargetResponses(context.Background(), target, []byte("{}"), "application/json", "session-test"); err == nil {
			t.Fatalf("OpenTargetResponses(%+v) accepted an invalid target", target)
		}
	}
}

func TestOpenTargetResponsesDistinguishesHeaderTimeoutFromDialFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)
	transport := &http.Transport{ResponseHeaderTimeout: 30 * time.Millisecond}
	client := mustClientWithHTTPClient(t, upstream.URL, &http.Client{Transport: transport})
	t.Cleanup(transport.CloseIdleConnections)

	_, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenCodeGoID,
	}, []byte("{}"), "application/json", "session-timeout-test")
	var headerTimeout providers.ResponseHeaderTimeoutError
	if !errors.As(err, &headerTimeout) {
		t.Fatalf("error = %v, want ResponseHeaderTimeoutError", err)
	}
	if strings.Contains(err.Error(), testCredential) {
		t.Fatal("timeout error leaked the provider credential")
	}

	listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
	if listenErr != nil {
		t.Fatalf("listen: %v", listenErr)
	}
	deadAddress := listener.Addr().String()
	if closeErr := listener.Close(); closeErr != nil {
		t.Fatalf("close listener: %v", closeErr)
	}
	dialClient := mustClient(t, newEnabledClientConfig("http://"+deadAddress))
	_, dialErr := dialClient.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenCodeGoID,
	}, []byte("{}"), "application/json", "session-dial-test")
	if dialErr == nil || errors.As(dialErr, &headerTimeout) {
		t.Fatalf("dial error = %v, want an unclassified transport failure", dialErr)
	}
}

func mustClientWithHTTPClient(t *testing.T, baseURL string, httpClient *http.Client) *providers.Client {
	t.Helper()
	client, err := providers.NewClient(newEnabledClientConfig(baseURL), httpClient)
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}
	return client
}
