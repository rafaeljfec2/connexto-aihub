package gateway_test

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func TestOpenRouterSSEIsProgressiveAndClientCancellationReachesUpstream(t *testing.T) {
	release := make(chan struct{})
	upstreamCanceled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+openRouterTestKey {
			t.Error("OpenRouter credential header mismatch")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"response.created\"}\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-release:
		}
		close(upstreamCanceled)
	}))
	defer upstream.Close()
	defer close(release)

	handler, _, _ := newRoutedResponsesHandler(t, "https://unused.example.test", upstream.URL)
	gatewayServer := httptest.NewServer(handler)
	defer gatewayServer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayServer.URL+"/v1/responses", strings.NewReader(`{"model":"openai/model-a","input":"hello","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(gateway.ProviderHeader, providers.OpenRouterID)
	response, err := gatewayServer.Client().Do(request)
	if err != nil {
		t.Fatalf("gateway request error = %v", err)
	}
	defer response.Body.Close()
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil || !strings.Contains(line, "response.created") {
		t.Fatalf("first SSE line = %q, error = %v", line, err)
	}
	cancel()
	select {
	case <-upstreamCanceled:
	case <-time.After(time.Second):
		t.Fatal("client cancellation did not reach the OpenRouter upstream")
	}
}
