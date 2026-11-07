package providers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/providers"
)

const (
	connextoProxyTestCursorKey  = "cursor-key-test-abc"
	connextoProxyTestGatewayKey = "test-gateway-key-12345"
)

func TestConnextoProxyResponsesPreservesBodyAndHeaders(t *testing.T) {
	payload := `{"model":"composer-2.5","input":"hello","stream":true}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
			t.Errorf("request = %s %s, want POST /v1/responses", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-Gateway-API-Key"); got != connextoProxyTestGatewayKey {
			t.Errorf("X-Gateway-API-Key = %q, want %q", got, connextoProxyTestGatewayKey)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+connextoProxyTestCursorKey {
			t.Errorf("Authorization = %q, want Bearer token", got)
		}
		if got := r.Header.Get("x-api-key"); got != connextoProxyTestCursorKey {
			t.Errorf("x-api-key = %q, want %q", got, connextoProxyTestCursorKey)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if string(body) != payload {
			t.Errorf("body = %q, want %q", body, payload)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	response, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.CursorID,
		ModelID:    "composer-2.5",
	}, []byte(payload), "application/json", "session-1")
	if err != nil {
		t.Fatalf("OpenTargetResponses() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}

func TestConnextoProxyFetchModelDescriptorsPreservesOriginalIDs(t *testing.T) {
	mockCatalog := `{"object":"list","data":[{"id":"composer-2.5","object":"model","created":1735689600,"owned_by":"cursor"},{"id":"grok-4.6","object":"model","created":1735689600,"owned_by":"cursor"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("request = %s %s, want GET /v1/models", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, mockCatalog)
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	descriptors, err := client.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}
	if len(descriptors) != 2 {
		t.Fatalf("descriptors count = %d, want 2", len(descriptors))
	}
	if descriptors[0].ID != "composer-2.5" || descriptors[1].ID != "grok-4.6" {
		t.Fatalf("model IDs mismatch: %+v", descriptors)
	}
	if descriptors[0].Name != "composer-2.5" {
		t.Fatalf("model name mismatch: %s", descriptors[0].Name)
	}
	if descriptors[0].ToolsSupported == nil || !*descriptors[0].ToolsSupported {
		t.Fatalf("ToolsSupported = %v, want explicit true", descriptors[0].ToolsSupported)
	}
	if descriptors[0].ReasoningSupported == nil || !*descriptors[0].ReasoningSupported {
		t.Fatalf("ReasoningSupported = %v, want explicit true", descriptors[0].ReasoningSupported)
	}
	if !descriptors[0].InputModalitiesPresent {
		t.Fatalf("InputModalitiesPresent = false, want true for E08 image_input routing")
	}
	if len(descriptors[0].InputModalities) == 0 || descriptors[0].InputModalities[0] != "text" {
		t.Fatalf("InputModalities = %v, want text and image/* defaults", descriptors[0].InputModalities)
	}
	foundImage := false
	for _, modality := range descriptors[0].InputModalities {
		if strings.HasPrefix(modality, "image/") {
			foundImage = true
			break
		}
	}
	if !foundImage {
		t.Fatalf("InputModalities = %v, want at least one image/* entry", descriptors[0].InputModalities)
	}
}

func TestConnextoProxyFetchModelDescriptorsHonorsUpstreamCapabilityFlags(t *testing.T) {
	mockCatalog := `{"object":"list","data":[{"id":"legacy-model","object":"model","capabilities":{"tools":{"supported":false},"reasoning":{"supported":true}}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, mockCatalog)
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	descriptors, err := client.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors count = %d, want 1", len(descriptors))
	}
	if descriptors[0].ToolsSupported == nil || *descriptors[0].ToolsSupported {
		t.Fatalf("ToolsSupported = %v, want explicit false from upstream", descriptors[0].ToolsSupported)
	}
	if descriptors[0].ReasoningSupported == nil || !*descriptors[0].ReasoningSupported {
		t.Fatalf("ReasoningSupported = %v, want explicit true from upstream", descriptors[0].ReasoningSupported)
	}
}

func TestConnextoProxyFetchModelDescriptorsHonorsUpstreamInputModalities(t *testing.T) {
	mockCatalog := `{"object":"list","data":[{"id":"text-only","object":"model","modalities":{"input":["text"]}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, mockCatalog)
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	descriptors, err := client.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors count = %d, want 1", len(descriptors))
	}
	if !descriptors[0].InputModalitiesPresent {
		t.Fatal("InputModalitiesPresent = false")
	}
	if len(descriptors[0].InputModalities) != 1 || descriptors[0].InputModalities[0] != "text" {
		t.Fatalf("InputModalities = %v, want upstream [text] only", descriptors[0].InputModalities)
	}
}

func TestConnextoProxyFetchModelDescriptorsRejectsInvalidPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":[{"object":"model"}]}`) // missing id
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	if _, err := client.FetchModelDescriptors(context.Background()); err == nil {
		t.Fatal("FetchModelDescriptors() accepted a model without an id")
	}
}

func TestConnextoProxyFetchModelDescriptorsRejectsDuplicates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":[{"id":"m1"},{"id":"m1"}]}`)
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	if _, err := client.FetchModelDescriptors(context.Background()); err == nil {
		t.Fatal("FetchModelDescriptors() accepted duplicate model IDs")
	}
}

func TestConnextoProxyResponsesPreservesUpstreamHTTPErrorAndSanitizes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `{"error":{"message":"secret cursor internal details"}}`)
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	response, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.CursorID,
		ModelID:    "composer-2.5",
	}, []byte(`{"model":"composer-2.5","input":"hello"}`), "application/json", "")
	if err != nil {
		t.Fatalf("OpenTargetResponses() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if strings.Contains(string(body), "secret cursor") {
		t.Fatalf("upstream error leaked internal details: %s", string(body))
	}
	if !strings.Contains(string(body), "Connexto Proxy request failed") {
		t.Fatalf("expected sanitized error body, got: %s", string(body))
	}
}

func TestConnextoProxyResponsesSanitizesPayloadTooLargeWithHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = io.WriteString(w, `{"error":{"message":"Request body is too large."}}`)
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	response, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.CursorID,
		ModelID:    "composer-2.5",
	}, []byte(`{"model":"composer-2.5","input":"hello"}`), "application/json", "")
	if err != nil {
		t.Fatalf("OpenTargetResponses() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	text := string(body)
	if !strings.Contains(text, "GATEWAY_JSON_BODY_LIMIT_BYTES") {
		t.Fatalf("expected payload-too-large hint, got: %s", text)
	}
	if strings.Contains(text, "Request body is too large") {
		t.Fatalf("upstream 413 body leaked: %s", text)
	}
}

func TestConnextoProxyResponsesMapsHeaderTimeoutWithoutCredential(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)

	transport := &http.Transport{ResponseHeaderTimeout: 20 * time.Millisecond}
	client := newConnextoProxyTestClient(t, server.URL+"/v1", &http.Client{Transport: transport}, connextoProxyTestCursorKey)
	_, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.CursorID,
	}, []byte(`{"model":"composer-2.5","input":"hello"}`), "application/json", "")
	var timeout providers.ResponseHeaderTimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("error = %v, want response header timeout", err)
	}
	if strings.Contains(err.Error(), connextoProxyTestCursorKey) {
		t.Fatal("timeout error exposed the credential")
	}
}

func TestConnextoProxyResponsesRejectsMissingCursorCredential(t *testing.T) {
	client, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:           "http://127.0.0.1:9090/v1",
		ManagementKey:     connextoProxyTestGatewayKey,
		Enabled:           true,
		LogicalProviderID: providers.CursorID,
		InferenceOptional: false,
	}, nil)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}
	_, err = client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.CursorID,
		ModelID:    "composer-2.5",
	}, []byte(`{"model":"composer-2.5","input":"hello"}`), "application/json", "")
	if !errors.Is(err, providers.ErrNoCredential) {
		t.Fatalf("OpenTargetResponses() error = %v, want ErrNoCredential", err)
	}
}

func TestConnextoProxyProbeEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"data":[{"id":"composer-2.5"}]}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if err := providers.ProbeConnextoProxyEndpoint(context.Background(), server.URL+"/v1", connextoProxyTestCursorKey); err != nil {
		t.Fatalf("ProbeConnextoProxyEndpoint() error = %v", err)
	}

	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer failingServer.Close()

	if err := providers.ProbeConnextoProxyEndpoint(context.Background(), failingServer.URL+"/v1", "wrong-key"); err == nil {
		t.Fatal("ProbeConnextoProxyEndpoint() accepted failing status")
	}
}

func TestConnextoProxyAntigravityEnabledWithAPIKeyLikeCursor(t *testing.T) {
	adapter, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:            "http://127.0.0.1:9090/v1",
		APIKey:             connextoProxyTestCursorKey,
		ManagementKey:      "",
		Enabled:            true,
		LogicalProviderID:  providers.AntigravityID,
		InferenceOptional:  true,
	}, nil)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}
	if !adapter.Enabled() {
		t.Fatal("Antigravity should be enabled when API key is set (same as Cursor pattern)")
	}
}

func TestConnextoProxyReconfigureAndEnabled(t *testing.T) {
	client := newConnextoProxyTestClient(t, "http://127.0.0.1:9090/v1", nil, connextoProxyTestCursorKey)
	if !client.Enabled() {
		t.Fatal("client should be enabled with credential")
	}

	// Reconfigure with empty credential
	if err := client.Reconfigure(providers.Settings{
		BaseURL:           "http://127.0.0.1:9090/v1",
		APIKey:            "",
		Enabled:           true,
		LogicalProviderID: providers.CursorID,
	}); err != nil {
		t.Fatalf("Reconfigure() error = %v", err)
	}
	if client.Enabled() {
		t.Fatal("client should not be enabled without credential")
	}

	// Reconfigure disabled
	if err := client.Reconfigure(providers.Settings{
		BaseURL:           "http://127.0.0.1:9090/v1",
		APIKey:            connextoProxyTestCursorKey,
		Enabled:           false,
		LogicalProviderID: providers.CursorID,
	}); err != nil {
		t.Fatalf("Reconfigure() error = %v", err)
	}
	if client.Enabled() {
		t.Fatal("client should not be enabled when enabled=false")
	}
}

func newConnextoProxyTestClient(t *testing.T, endpoint string, client *http.Client, credential string) *providers.ConnextoProxyClient {
	t.Helper()
	adapter, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:            endpoint,
		APIKey:             credential,
		ManagementKey:      connextoProxyTestGatewayKey,
		Enabled:            true,
		LogicalProviderID:  providers.CursorID,
		InferenceOptional:  false,
	}, client)
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}
	return adapter
}

func TestConnextoProxyFetchModelDescriptorsCursorExcludesAntigravity(t *testing.T) {
	mockCatalog := `{"object":"list","data":[` +
		`{"id":"composer-2.5","owned_by":"cursor"},` +
		`{"id":"antigravity:gemini-3.8-flash","owned_by":"antigravity"}` +
		`]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, mockCatalog)
	}))
	defer server.Close()

	client := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	descriptors, err := client.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}
	if len(descriptors) != 1 || descriptors[0].ID != "composer-2.5" {
		t.Fatalf("got %+v, want only composer-2.5", descriptors)
	}
}

func TestConnextoProxyFetchModelDescriptorsAntigravityOnlyNamespace(t *testing.T) {
	mockCatalog := `{"object":"list","data":[` +
		`{"id":"composer-2.5","owned_by":"cursor"},` +
		`{"id":"antigravity:gemini-3.8-flash","owned_by":"antigravity","name":"Antigravity Gemini"}` +
		`]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, mockCatalog)
	}))
	defer server.Close()

	adapter, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:            server.URL + "/v1",
		ManagementKey:      connextoProxyTestGatewayKey,
		Enabled:            true,
		LogicalProviderID:  providers.AntigravityID,
		InferenceOptional:  true,
	}, server.Client())
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}
	descriptors, err := adapter.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}
	if len(descriptors) != 1 || descriptors[0].ID != "antigravity:gemini-3.8-flash" {
		t.Fatalf("got %+v, want only antigravity model", descriptors)
	}
	if descriptors[0].ToolsSupported != nil && *descriptors[0].ToolsSupported {
		t.Fatal("Antigravity without capability flags should not claim tools")
	}
}

func TestConnextoProxyAntigravityResponsesOmitsCursorHeaders(t *testing.T) {
	payload := `{"model":"antigravity:gemini-3.8-flash","input":"hello"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("x-api-key") != "" {
			t.Error("Antigravity request must not send Cursor API key headers")
		}
		if got := r.Header.Get("X-Gateway-API-Key"); got != connextoProxyTestGatewayKey {
			t.Errorf("X-Gateway-API-Key = %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	adapter, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:            server.URL + "/v1",
		APIKey:             "",
		ManagementKey:      connextoProxyTestGatewayKey,
		Enabled:            true,
		LogicalProviderID:  providers.AntigravityID,
		InferenceOptional:  true,
	}, server.Client())
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}
	resp, err := adapter.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.AntigravityID,
		ModelID:    "antigravity:gemini-3.8-flash",
	}, []byte(payload), "application/json", "")
	if err != nil {
		t.Fatalf("OpenTargetResponses() error = %v", err)
	}
	_ = resp.Body.Close()
}

