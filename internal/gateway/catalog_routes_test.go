package gateway_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/providers"
	"aihub/internal/server"
)

func TestCatalogRoutesThroughRouter(t *testing.T) {
	descriptor := providers.ModelDescriptor{
		ID:                      "vendor/model",
		Name:                    "Vendor Model",
		InputModalitiesPresent:  true,
		OutputModalities:        []string{"text"},
		OutputModalitiesPresent: true,
	}

	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return []providers.ModelDescriptor{descriptor}, nil
	}, providers.StateHealthy)

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	logger, _ := newTestLogger()
	readiness := health.NewRegistry()

	router := server.NewRouter(logger, health.Live(), readiness.Ready(),
		server.Route{Method: http.MethodGet, Pattern: "/v1/catalog/models", Handler: gateway.NewCatalogModelsHandler(manager)},
		server.Route{Method: http.MethodGet, Pattern: "/v1/catalog/models/{provider}/{model...}", Handler: gateway.NewCatalogModelHandler(manager)},
		server.Route{Method: http.MethodGet, Pattern: "/v1/catalog/providers", Handler: gateway.NewCatalogProvidersHandler(manager)},
		server.Route{Method: http.MethodPost, Pattern: "/v1/catalog/sync", Handler: gateway.NewCatalogSyncHandler(manager)},
	)

	detail := httptest.NewRequest(http.MethodGet, "/v1/catalog/models/opencode-go/vendor/model", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, detail)

	if recorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}

	var body struct {
		ModelID string `json:"model_id"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode detail: %v", err)
	}

	if body.ModelID != "vendor/model" {
		t.Fatalf("model id = %q, want vendor/model", body.ModelID)
	}

	for _, target := range []string{"/readyz", "/v1/catalog/models", "/v1/catalog/providers"} {
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", target, recorder.Code)
		}
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"opencode-go"}`)))

	if recorder.Code != http.StatusOK {
		t.Fatalf("sync status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
}

func TestCatalogRoutesStayHealthyWithoutProvider(t *testing.T) {
	manager := newGatewayCatalogManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return nil, nil
	}, providers.StateDisabled)

	logger, _ := newTestLogger()
	readiness := health.NewRegistry()

	router := server.NewRouter(logger, health.Live(), readiness.Ready(),
		server.Route{Method: http.MethodGet, Pattern: "/v1/catalog/models", Handler: gateway.NewCatalogModelsHandler(manager)},
		server.Route{Method: http.MethodPost, Pattern: "/v1/catalog/sync", Handler: gateway.NewCatalogSyncHandler(manager)},
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("readyz status = %d, want 200", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/catalog/sync", strings.NewReader(`{"provider":"opencode-go"}`)))

	if recorder.Code != http.StatusConflict {
		t.Fatalf("sync status = %d, want 409 (body %s)", recorder.Code, recorder.Body.String())
	}
}
