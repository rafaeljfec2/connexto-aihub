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
		LogicalProviderID: providers.CursorID,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	registry := providers.NewRegistry()
	registry.Bind(providers.CursorID, client)

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
	request.Header.Set(gateway.ProviderHeader, providers.CursorID)

	response, err := gatewayServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.StatusCode)
	}
}

func TestCursorIntelligentRoutingCodexLikePayload(t *testing.T) {
	mockCatalog := `{"object":"list","data":[{"id":"composer-2.5","object":"model","owned_by":"cursor"}]}`
	var responsesHit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, mockCatalog)
		case "/v1/responses":
			responsesHit = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":"resp-routed"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	proxyClient, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:            server.URL + "/v1",
		APIKey:             connextoProxyTestKey,
		ManagementKey:      connextoProxyTestKey,
		Enabled:            true,
		LogicalProviderID:  providers.CursorID,
	}, nil)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}

	registry := providers.NewRegistry()
	registry.Bind(providers.CursorID, proxyClient)

	catalogManager := gatewayTestCatalog(t, time.Now(), map[string]catalog.SourceFunc{
		providers.CursorID: proxyClient.FetchModelDescriptors,
	})

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		}, time.Minute)

	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{
				{ID: providers.CursorID, Enabled: true, State: providers.StateHealthy},
			}
		},
		StaleAfter: 24 * time.Hour,
	}, nil, nil, nil)

	gatewayServer := httptest.NewServer(handler)
	defer gatewayServer.Close()

	codexLikeBody := `{"model":"composer-2.5","input":"hello","tools":[{"type":"function","name":"noop","parameters":{"type":"object","properties":{}}}],"reasoning":{"effort":"medium"}}`
	request, err := http.NewRequest(http.MethodPost, gatewayServer.URL+"/v1/responses", strings.NewReader(codexLikeBody))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := gatewayServer.Client().Do(request)
	if err != nil {
		t.Fatalf("gateway request error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d body = %s, want 200 (routing should reach execution)", response.StatusCode, string(body))
	}
	if !responsesHit {
		t.Fatal("upstream /v1/responses was not called")
	}
}

func TestCursorIntelligentRoutingImageInputPayload(t *testing.T) {
	mockCatalog := `{"object":"list","data":[{"id":"composer-2.5","object":"model","owned_by":"cursor"}]}`
	var responsesHit bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, mockCatalog)
		case "/v1/responses":
			responsesHit = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":"resp-multimodal"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	proxyClient, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:            server.URL + "/v1",
		APIKey:             connextoProxyTestKey,
		ManagementKey:      connextoProxyTestKey,
		Enabled:            true,
		LogicalProviderID:  providers.CursorID,
	}, nil)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}

	registry := providers.NewRegistry()
	registry.Bind(providers.CursorID, proxyClient)

	catalogManager := gatewayTestCatalog(t, time.Now(), map[string]catalog.SourceFunc{
		providers.CursorID: proxyClient.FetchModelDescriptors,
	})

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		}, time.Minute)

	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{
				{ID: providers.CursorID, Enabled: true, State: providers.StateHealthy},
			}
		},
		StaleAfter: 24 * time.Hour,
	}, nil, nil, nil)

	gatewayServer := httptest.NewServer(handler)
	defer gatewayServer.Close()

	png := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	body := strings.Join([]string{
		`{"model":"composer-2.5","input":[`,
		`{"type":"input_text","text":"describe"},`,
		`{"type":"input_image","image_url":"data:image/png;base64,`, png, `"}`,
		`],"tools":[{"type":"function","name":"noop","parameters":{"type":"object","properties":{}}}],`,
		`"reasoning":{"effort":"medium"}}`,
	}, "")
	request, err := http.NewRequest(http.MethodPost, gatewayServer.URL+"/v1/responses", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := gatewayServer.Client().Do(request)
	if err != nil {
		t.Fatalf("gateway request error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d body = %s, want 200 (image_input must not block routing)", response.StatusCode, string(raw))
	}
	if !responsesHit {
		t.Fatal("upstream /v1/responses was not called")
	}
}

func TestAntigravityExplicitProviderRouting(t *testing.T) {
	var sawAntigravity bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"object":"list","data":[`+
				`{"id":"composer-2.5","owned_by":"cursor"},`+
				`{"id":"antigravity:gemini-3.8-flash","owned_by":"antigravity"}`+
				`]}`)
		case "/v1/responses":
			sawAntigravity = true
			if r.Header.Get("Authorization") != "" || r.Header.Get("x-api-key") != "" {
				t.Error("Antigravity upstream must not receive Cursor API key headers")
			}
			if got := r.Header.Get("X-Gateway-API-Key"); got != connextoProxyTestKey {
				t.Errorf("X-Gateway-API-Key = %q, want %q", got, connextoProxyTestKey)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id":"resp-antigravity"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	transport := providers.NewConnextoProxyHTTPClient()
	cursorClient, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:            server.URL + "/v1",
		APIKey:             connextoProxyTestKey,
		ManagementKey:      connextoProxyTestKey,
		Enabled:            true,
		LogicalProviderID:  providers.CursorID,
	}, transport)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() cursor error = %v", err)
	}
	agClient, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:            server.URL + "/v1",
		ManagementKey:      connextoProxyTestKey,
		Enabled:            true,
		LogicalProviderID:  providers.AntigravityID,
		InferenceOptional:  true,
	}, transport)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() antigravity error = %v", err)
	}

	registry := providers.NewRegistry()
	registry.Bind(providers.CursorID, cursorClient)
	registry.Bind(providers.AntigravityID, agClient)

	now := time.Now()
	catalogManager := gatewayTestCatalog(t, now, map[string]catalog.SourceFunc{
		providers.CursorID:     cursorClient.FetchModelDescriptors,
		providers.AntigravityID: agClient.FetchModelDescriptors,
	})

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{ProviderID: target.ProviderID, ModelID: target.ModelID}, payload, contentType, sessionID)
		}, time.Minute)

	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{
				{ID: providers.CursorID, Enabled: true, State: providers.StateHealthy},
				{ID: providers.AntigravityID, Enabled: true, State: providers.StateHealthy},
			}
		},
		StaleAfter: 24 * time.Hour,
	}, nil, nil, nil)

	gatewayServer := httptest.NewServer(handler)
	defer gatewayServer.Close()

	body := `{"model":"antigravity:gemini-3.8-flash","input":"hello"}`
	request, err := http.NewRequest(http.MethodPost, gatewayServer.URL+"/v1/responses", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(gateway.ProviderHeader, providers.AntigravityID)

	response, err := gatewayServer.Client().Do(request)
	if err != nil {
		t.Fatalf("gateway request error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d body = %s", response.StatusCode, string(raw))
	}
	if !sawAntigravity {
		t.Fatal("upstream /v1/responses was not invoked for Antigravity")
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
		BaseURL:            proxyEndpoint + "/v1",
		APIKey:             connextoProxyTestKey,
		ManagementKey:      connextoProxyTestKey,
		Enabled:            true,
		LogicalProviderID: providers.CursorID,
	}, nil)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}

	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.CursorID, proxyClient)

	catalogManager := gatewayTestCatalog(t, time.Now(), map[string]catalog.SourceFunc{
		providers.CursorID: func(context.Context) ([]providers.ModelDescriptor, error) {
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
			}
		},
		StaleAfter: 24 * time.Hour,
	}, nil, nil, nil)
}

