package providers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/providers"
)

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
		BaseURL:           server.URL + "/v1",
		ManagementKey:     connextoProxyTestGatewayKey,
		Enabled:           true,
		LogicalProviderID: providers.AntigravityID,
		InferenceOptional: true,
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

func TestConnextoProxyFetchModelDescriptorsAssignsLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"data": [
				{"id": "composer-2.5", "object": "model", "owned_by": "cursor"},
				{"id": "antigravity:gemini-3.8-flash", "object": "model", "owned_by": "antigravity"}
			]
		}`)
	}))
	defer server.Close()

	cursorClient := newConnextoProxyTestClient(t, server.URL+"/v1", server.Client(), connextoProxyTestCursorKey)
	cursorDescriptors, err := cursorClient.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors(cursor) error = %v", err)
	}
	if len(cursorDescriptors) != 1 || cursorDescriptors[0].ID != "composer-2.5" {
		t.Fatalf("cursorDescriptors = %+v", cursorDescriptors)
	}
	if cursorDescriptors[0].LimitContext == nil || *cursorDescriptors[0].LimitContext != 200000 {
		t.Fatalf("composer-2.5 LimitContext = %v, want 200000", cursorDescriptors[0].LimitContext)
	}
	if cursorDescriptors[0].LimitOutput == nil || *cursorDescriptors[0].LimitOutput != 16384 {
		t.Fatalf("composer-2.5 LimitOutput = %v, want 16384", cursorDescriptors[0].LimitOutput)
	}
	if cursorDescriptors[0].StructuredOutput == nil || !*cursorDescriptors[0].StructuredOutput {
		t.Fatalf("composer-2.5 StructuredOutput = %v, want true", cursorDescriptors[0].StructuredOutput)
	}

	antigravityClient, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:           server.URL + "/v1",
		APIKey:            "",
		ManagementKey:     connextoProxyTestGatewayKey,
		Enabled:           true,
		LogicalProviderID: providers.AntigravityID,
		InferenceOptional: true,
	}, server.Client())
	if err != nil {
		t.Fatalf("NewConnextoProxyClient() error = %v", err)
	}
	antigravityDescriptors, err := antigravityClient.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors(antigravity) error = %v", err)
	}
	if len(antigravityDescriptors) != 1 || antigravityDescriptors[0].ID != "antigravity:gemini-3.8-flash" {
		t.Fatalf("antigravityDescriptors = %+v", antigravityDescriptors)
	}
	if antigravityDescriptors[0].LimitContext == nil || *antigravityDescriptors[0].LimitContext != 1048576 {
		t.Fatalf("gemini-3.8-flash LimitContext = %v, want 1048576", antigravityDescriptors[0].LimitContext)
	}
	if antigravityDescriptors[0].LimitOutput == nil || *antigravityDescriptors[0].LimitOutput != 65536 {
		t.Fatalf("gemini-3.8-flash LimitOutput = %v, want 65536", antigravityDescriptors[0].LimitOutput)
	}
}

