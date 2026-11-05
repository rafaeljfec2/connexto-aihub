package gateway_test

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

const connextoProxyTestKey = "connexto-gw-test-key"

func TestConnextoProxyResponsesRoutingAndStreaming(t *testing.T) {
	upstreamCanceled := make(chan struct{})
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Gateway-API-Key") != connextoProxyTestKey {
			t.Errorf("X-Gateway-API-Key = %q, want %q", r.Header.Get("X-Gateway-API-Key"), connextoProxyTestKey)
		}
		if r.Header.Get("Authorization") != "Bearer "+connextoProxyTestKey {
			t.Errorf("Authorization = %q, want Bearer token", r.Header.Get("Authorization"))
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

	handler := newConnextoProxyRoutedHandler(t, upstream.URL)
	gatewayServer := httptest.NewServer(handler)
	defer gatewayServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayServer.URL+"/v1/responses",
		strings.NewReader(`{"model":"composer-2.5","input":"hello","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(gateway.ProviderHeader, providers.ConnextoProxyID)

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
		t.Fatal("client cancellation did not reach the Connexto Proxy upstream")
	}
}

func TestCursorResponsesRoutingAndStreaming(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Gateway-API-Key") != connextoProxyTestKey {
			t.Errorf("X-Gateway-API-Key = %q, want %q", r.Header.Get("X-Gateway-API-Key"), connextoProxyTestKey)
		}
		if r.Header.Get("Authorization") != "Bearer "+connextoProxyTestKey {
			t.Errorf("Authorization = %q, want Bearer token", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"response.created\"}\n\n")
		w.(http.Flusher).Flush()
	}))
	defer upstream.Close()

	handler := newConnextoProxyRoutedHandler(t, upstream.URL)
	gatewayServer := httptest.NewServer(handler)
	defer gatewayServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayServer.URL+"/v1/responses",
		strings.NewReader(`{"model":"composer-2.5","input":"hello from cursor","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(gateway.ProviderHeader, providers.CursorID)

	response, err := gatewayServer.Client().Do(request)
	if err != nil {
		t.Fatalf("gateway request error = %v", err)
	}
	defer response.Body.Close()

	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil || !strings.Contains(line, "response.created") {
		t.Fatalf("first SSE line = %q, error = %v", line, err)
	}
}

func TestConnextoProxyUnavailableWhenDisabled(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL: upstream.URL + "/v1",
		APIKey:  connextoProxyTestKey,
		Enabled: false, // disabled
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	registry := providers.NewRegistry()
	registry.Bind(providers.ConnextoProxyID, client)

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		}, time.Minute)

	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, nil, nil, nil, nil)
	gatewayServer := httptest.NewServer(handler)
	defer gatewayServer.Close()

	request, err := http.NewRequest(http.MethodPost, gatewayServer.URL+"/v1/responses",
		strings.NewReader(`{"model":"composer-2.5","input":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(gateway.ProviderHeader, providers.ConnextoProxyID)

	response, err := gatewayServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.StatusCode)
	}
}

func newConnextoProxyRoutedHandler(t *testing.T, proxyEndpoint string) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	openCode, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: "http://127.0.0.1:8099", APIKey: "unused"}, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	proxyClient, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:       proxyEndpoint + "/v1",
		APIKey:        connextoProxyTestKey,
		ManagementKey: connextoProxyTestKey,
		Enabled:       true,
	}, nil)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}

	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.CursorID, proxyClient)
	registry.Bind(providers.ConnextoProxyID, proxyClient)

	catalogManager := gatewayTestCatalog(t, time.Now(), map[string]catalog.SourceFunc{
		providers.CursorID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{gatewayTestDescriptor("composer-2.5")}, nil
		},
		providers.ConnextoProxyID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{gatewayTestDescriptor("composer-2.5")}, nil
		},
	})

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		}, time.Minute)

	return gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{
				{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy},
				{ID: providers.CursorID, Enabled: true, State: providers.StateHealthy},
				{ID: providers.ConnextoProxyID, Enabled: true, State: providers.StateHealthy},
			}
		},
		StaleAfter: 24 * time.Hour,
	}, nil, nil, nil)
}

