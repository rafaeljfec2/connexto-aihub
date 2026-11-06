package gateway_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

var gatewayCatalogNow = time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)

func gatewayDescriptor(id string, image, tools bool) providers.ModelDescriptor {
	input := []string{"text"}
	if image {
		input = append(input, "image/png")
	}

	return providers.ModelDescriptor{
		ID:                      id,
		Name:                    id,
		InputModalities:         input,
		InputModalitiesPresent:  true,
		OutputModalities:        []string{"text"},
		OutputModalitiesPresent: true,
		ToolsSupported:          gatewayBoolPtr(tools),
		Raw:                     json.RawMessage(`{"id":"` + id + `"}`),
	}
}

func gatewayBoolPtr(value bool) *bool {
	return &value
}

func newGatewayCatalogManager(t *testing.T, source catalog.SourceFunc, state providers.State) *catalog.Manager {
	t.Helper()

	logger, _ := newTestLogger()
	store := catalog.NewStore(t.TempDir(), logger)

	manager := catalog.NewManager(logger, func() []providers.Status {
		return []providers.Status{{
			ID:      providers.OpenCodeGoID,
			Enabled: state != providers.StateDisabled,
			State:   state,
		}}
	}, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: source,
	}, store, catalog.Options{
		StaleAfter:  time.Hour,
		SyncTimeout: time.Second,
		Now:         func() time.Time { return gatewayCatalogNow },
	})

	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	return manager
}

func serveCatalog(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func TestCatalogModelsHandlerListsNormalizedModels(t *testing.T) {
	descriptors := []providers.ModelDescriptor{
		gatewayDescriptor("A", true, true),
		gatewayDescriptor("B", false, false),
	}

	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return descriptors, nil
	}, providers.StateHealthy)

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	recorder := serveCatalog(
		gateway.NewCatalogModelsHandler(manager),
		httptest.NewRequest(http.MethodGet, "/v1/catalog/models", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}

	var body struct {
		Object string `json:"object"`
		Data   []struct {
			Provider              string `json:"provider"`
			ModelID               string `json:"model_id"`
			ListingState          string `json:"listing_state"`
			ExecutionVerification string `json:"execution_verification"`
			ProviderMetadata      string `json:"provider_metadata"`
			Pricing               *struct {
				Input    *float64 `json:"input"`
				Complete bool     `json:"complete"`
			} `json:"pricing"`
			Capabilities map[string]struct {
				State string `json:"state"`
			} `json:"capabilities"`
		} `json:"data"`
		Meta struct {
			Count int `json:"count"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Object != "list" || body.Meta.Count != 2 || len(body.Data) != 2 {
		t.Fatalf("response shape = %+v", body)
	}

	if body.Data[0].ModelID != "A" || body.Data[1].ModelID != "B" {
		t.Fatalf("order = %s/%s, want A/B", body.Data[0].ModelID, body.Data[1].ModelID)
	}

	if body.Data[0].ListingState != "current" {
		t.Fatalf("listing state = %q, want current", body.Data[0].ListingState)
	}

	if body.Data[0].ExecutionVerification != "unverified" {
		t.Fatalf("execution verification = %q, want unverified", body.Data[0].ExecutionVerification)
	}

	if body.Data[0].Capabilities["image_input"].State != "supported" {
		t.Fatalf("image_input state = %q", body.Data[0].Capabilities["image_input"].State)
	}

	if body.Data[0].ProviderMetadata != "" {
		t.Fatal("list response included provider_metadata")
	}

	if body.Data[0].Pricing != nil {
		t.Fatalf("list response included pricing without provider cost metadata: %+v", body.Data[0].Pricing)
	}
}

func TestCatalogModelsHandlerListsProjectedPricing(t *testing.T) {
	raw := json.RawMessage(`{
		"id":"deepseek-v4.1-flash",
		"cost":{"text":{"input":{"price":0.15,"cache":{"read":0.003}},"output":{"price":0.6}}}
	}`)
	descriptor := gatewayDescriptor("deepseek-v4.1-flash", false, true)
	descriptor.Raw = raw

	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return []providers.ModelDescriptor{descriptor}, nil
	}, providers.StateHealthy)

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	recorder := serveCatalog(
		gateway.NewCatalogModelsHandler(manager),
		httptest.NewRequest(http.MethodGet, "/v1/catalog/models", nil),
	)

	var body struct {
		Data []struct {
			Pricing *struct {
				Input      *float64 `json:"input"`
				Output     *float64 `json:"output"`
				CacheRead  *float64 `json:"cache_read"`
				Unit       string   `json:"unit"`
				Complete   bool     `json:"complete"`
			} `json:"pricing"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].Pricing == nil {
		t.Fatalf("pricing missing: %+v", body.Data)
	}
	if body.Data[0].Pricing.Input == nil || *body.Data[0].Pricing.Input != 0.15 {
		t.Fatalf("input = %v", body.Data[0].Pricing.Input)
	}
	if body.Data[0].Pricing.Output == nil || *body.Data[0].Pricing.Output != 0.6 {
		t.Fatalf("output = %v", body.Data[0].Pricing.Output)
	}
	if body.Data[0].Pricing.CacheRead == nil || *body.Data[0].Pricing.CacheRead != 0.003 {
		t.Fatalf("cache_read = %v", body.Data[0].Pricing.CacheRead)
	}
	if body.Data[0].Pricing.Unit != "per_1m_tokens" || !body.Data[0].Pricing.Complete {
		t.Fatalf("pricing meta = %+v", body.Data[0].Pricing)
	}
}

func TestCatalogModelsHandlerFiltersAndValidation(t *testing.T) {
	var current []providers.ModelDescriptor

	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}, providers.StateHealthy)

	current = []providers.ModelDescriptor{
		gatewayDescriptor("A", true, true),
		gatewayDescriptor("B", false, false),
		gatewayDescriptor("C", false, true),
	}

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	current = []providers.ModelDescriptor{
		gatewayDescriptor("A", true, true),
		gatewayDescriptor("B", false, false),
	}

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}

	handler := gateway.NewCatalogModelsHandler(manager)

	cases := []struct {
		name      string
		query     string
		wantCode  int
		wantCount int
	}{
		{name: "missing", query: "?presence=missing", wantCode: 200, wantCount: 1},
		{name: "current", query: "?listing_state=current", wantCode: 200, wantCount: 2},
		{name: "capability", query: "?capability=image_input", wantCode: 200, wantCount: 1},
		{name: "invalid presence", query: "?presence=bogus", wantCode: 400},
		{name: "invalid listing state", query: "?listing_state=bogus", wantCode: 400},
		{name: "invalid capability", query: "?capability=bogus", wantCode: 400},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serveCatalog(handler, httptest.NewRequest(http.MethodGet, "/v1/catalog/models"+testCase.query, nil))
			if recorder.Code != testCase.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", recorder.Code, testCase.wantCode, recorder.Body.String())
			}

			if testCase.wantCode != http.StatusOK {
				if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
					t.Fatalf("error Cache-Control = %q, want no-store", got)
				}

				return
			}

			var body struct {
				Meta struct {
					Count int `json:"count"`
				} `json:"meta"`
			}

			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if body.Meta.Count != testCase.wantCount {
				t.Fatalf("count = %d, want %d", body.Meta.Count, testCase.wantCount)
			}
		})
	}
}

func TestCatalogModelHandlerDetail(t *testing.T) {
	descriptors := []providers.ModelDescriptor{gatewayDescriptor("A", true, true)}

	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return descriptors, nil
	}, providers.StateHealthy)

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	handler := gateway.NewCatalogModelHandler(manager)

	request := httptest.NewRequest(http.MethodGet, "/v1/catalog/models/opencode-go/A", nil)
	request.SetPathValue("provider", providers.OpenCodeGoID)
	request.SetPathValue("model", "A")

	recorder := serveCatalog(handler, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}

	var body struct {
		ModelID          string          `json:"model_id"`
		ProviderMetadata json.RawMessage `json:"provider_metadata"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.ModelID != "A" || len(body.ProviderMetadata) == 0 {
		t.Fatalf("detail body = %+v", body)
	}

	unknownModel := httptest.NewRequest(http.MethodGet, "/v1/catalog/models/opencode-go/missing", nil)
	unknownModel.SetPathValue("provider", providers.OpenCodeGoID)
	unknownModel.SetPathValue("model", "missing")

	recorder = serveCatalog(handler, unknownModel)
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "model_not_found") {
		t.Fatalf("unknown model response = %d %s", recorder.Code, recorder.Body.String())
	}

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("error Cache-Control = %q, want no-store", got)
	}

	unknownProvider := httptest.NewRequest(http.MethodGet, "/v1/catalog/models/other/A", nil)
	unknownProvider.SetPathValue("provider", "other")
	unknownProvider.SetPathValue("model", "A")

	recorder = serveCatalog(handler, unknownProvider)
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "provider_not_found") {
		t.Fatalf("unknown provider response = %d %s", recorder.Code, recorder.Body.String())
	}

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("error Cache-Control = %q, want no-store", got)
	}
}

func TestCatalogProvidersHandler(t *testing.T) {
	var current []providers.ModelDescriptor

	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}, providers.StateHealthy)

	current = []providers.ModelDescriptor{gatewayDescriptor("A", false, false), gatewayDescriptor("B", false, false), gatewayDescriptor("C", false, false)}

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	current = []providers.ModelDescriptor{gatewayDescriptor("A", false, false), gatewayDescriptor("B", false, false)}

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}

	recorder := serveCatalog(
		gateway.NewCatalogProvidersHandler(manager),
		httptest.NewRequest(http.MethodGet, "/v1/catalog/providers", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var body struct {
		Data []struct {
			Provider      string `json:"provider"`
			ProviderState string `json:"provider_state"`
			Freshness     string `json:"freshness"`
			Sync          struct {
				Outcome string `json:"outcome"`
			} `json:"sync"`
			Models struct {
				Known   int `json:"known"`
				Present int `json:"present"`
				Missing int `json:"missing"`
			} `json:"models"`
		} `json:"data"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Data) != 1 {
		t.Fatalf("provider count = %d, want 1", len(body.Data))
	}

	provider := body.Data[0]
	if provider.Provider != providers.OpenCodeGoID || provider.ProviderState != "healthy" || provider.Freshness != "fresh" {
		t.Fatalf("provider = %+v", provider)
	}

	if provider.Sync.Outcome != "success" {
		t.Fatalf("outcome = %q, want success", provider.Sync.Outcome)
	}

	if provider.Models.Known != 3 || provider.Models.Present != 2 || provider.Models.Missing != 1 {
		t.Fatalf("models = %+v", provider.Models)
	}

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestCatalogSyncHandlerSuccess(t *testing.T) {
	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return []providers.ModelDescriptor{gatewayDescriptor("A", false, false)}, nil
	}, providers.StateHealthy)

	handler := gateway.NewCatalogSyncHandler(manager)

	request := httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"opencode-go"}`))
	recorder := serveCatalog(handler, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Providers []struct {
			Provider   string `json:"provider"`
			Outcome    string `json:"outcome"`
			DurationMS int64  `json:"duration_ms"`
			Counts     struct {
				Discovered int `json:"discovered"`
				Added      int `json:"added"`
			} `json:"counts"`
		} `json:"providers"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Providers) != 1 || body.Providers[0].Outcome != "success" {
		t.Fatalf("response = %+v", body)
	}

	if body.Providers[0].Counts.Discovered != 1 || body.Providers[0].Counts.Added != 1 {
		t.Fatalf("counts = %+v", body.Providers[0].Counts)
	}

	empty := httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", nil)
	recorder = serveCatalog(handler, empty)
	if recorder.Code != http.StatusOK {
		t.Fatalf("empty body status = %d, want 200", recorder.Code)
	}
}

func TestCatalogSyncHandlerErrors(t *testing.T) {
	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return nil, &providers.UpstreamHTTPError{StatusCode: 500}
	}, providers.StateHealthy)

	handler := gateway.NewCatalogSyncHandler(manager)

	recorder := serveCatalog(handler, httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"other"}`)))
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "provider_not_found") {
		t.Fatalf("unknown provider = %d %s", recorder.Code, recorder.Body.String())
	}

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("error Cache-Control = %q, want no-store", got)
	}

	recorder = serveCatalog(handler, httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"unknown":true}`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, want 400", recorder.Code)
	}

	recorder = serveCatalog(handler, httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"opencode-go"} {}`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("trailing body status = %d, want 400", recorder.Code)
	}

	recorder = serveCatalog(handler, httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"opencode-go"}`)))
	if recorder.Code != http.StatusBadGateway || !strings.Contains(recorder.Body.String(), "upstream_unavailable") {
		t.Fatalf("upstream failure = %d %s", recorder.Code, recorder.Body.String())
	}

	disabled := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return nil, nil
	}, providers.StateDisabled)

	recorder = serveCatalog(gateway.NewCatalogSyncHandler(disabled), httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"opencode-go"}`)))
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), "provider_disabled") {
		t.Fatalf("disabled provider = %d %s", recorder.Code, recorder.Body.String())
	}

	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("conflict Cache-Control = %q, want no-store", got)
	}
}

func TestCatalogSyncHandlerPersistenceFailure(t *testing.T) {
	logger, _ := newTestLogger()
	blockedDir := filepath.Join(t.TempDir(), "blocked")

	if err := os.WriteFile(blockedDir, []byte("blocked"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	store := catalog.NewStore(blockedDir, logger)
	manager := catalog.NewManager(logger, func() []providers.Status {
		return []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy}}
	}, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{gatewayDescriptor("A", false, false)}, nil
		},
	}, store, catalog.Options{
		StaleAfter:  time.Hour,
		SyncTimeout: time.Second,
		Now:         func() time.Time { return gatewayCatalogNow },
	})

	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	recorder := serveCatalog(
		gateway.NewCatalogSyncHandler(manager),
		httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"opencode-go"}`)),
	)

	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "internal_error") {
		t.Fatalf("persistence failure = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCatalogSyncHandlerSingleFlight(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})

	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		close(started)
		<-release

		return nil, errors.New("released")
	}, providers.StateHealthy)

	done := make(chan struct{})

	go func() {
		_, _ = manager.Sync(context.Background(), providers.OpenCodeGoID)
		close(done)
	}()

	<-started

	recorder := serveCatalog(
		gateway.NewCatalogSyncHandler(manager),
		httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"opencode-go"}`)),
	)

	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), "sync_in_progress") {
		t.Fatalf("single flight = %d %s", recorder.Code, recorder.Body.String())
	}

	close(release)
	<-done
}

func TestCatalogMissingSinceOnlyInDetail(t *testing.T) {
	var current []providers.ModelDescriptor

	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}, providers.StateHealthy)

	current = []providers.ModelDescriptor{gatewayDescriptor("A", false, false), gatewayDescriptor("B", false, false)}
	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	current = []providers.ModelDescriptor{gatewayDescriptor("A", false, false)}
	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}

	recorder := serveCatalog(
		gateway.NewCatalogModelsHandler(manager),
		httptest.NewRequest(http.MethodGet, "/v1/catalog/models?presence=missing", nil),
	)

	var listBody struct {
		Data []map[string]json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}

	if len(listBody.Data) != 1 {
		t.Fatalf("missing list count = %d, want 1", len(listBody.Data))
	}

	if _, ok := listBody.Data[0]["missing_since"]; ok {
		t.Fatal("list response included missing_since")
	}

	for _, forbidden := range []string{"enabled", "available", "ready"} {
		if _, ok := listBody.Data[0][forbidden]; ok {
			t.Fatalf("list response included forbidden field %q", forbidden)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/catalog/models/opencode-go/B", nil)
	request.SetPathValue("provider", providers.OpenCodeGoID)
	request.SetPathValue("model", "B")

	recorder = serveCatalog(gateway.NewCatalogModelHandler(manager), request)

	var detailBody map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &detailBody); err != nil {
		t.Fatalf("decode detail: %v", err)
	}

	if _, ok := detailBody["missing_since"]; !ok {
		t.Fatal("detail response omitted missing_since")
	}

	for _, forbidden := range []string{"enabled", "available", "ready"} {
		if _, ok := detailBody[forbidden]; ok {
			t.Fatalf("detail response included forbidden field %q", forbidden)
		}
	}
}
