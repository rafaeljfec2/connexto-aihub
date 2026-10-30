package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/health"
	"aihub/internal/server"
)

type markerHandler struct {
	requested *bool
}

func (m markerHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	*m.requested = true
}

func TestRouterRegistersAdditionalRoutes(t *testing.T) {
	invoked := new(bool)

	handler := server.NewRouter(
		testLogger(new(bytes.Buffer)),
		// health handlers unused in this scenario
		health.Live(),
		health.NewRegistry().Ready(),
		server.Route{
			Method:  http.MethodPost,
			Pattern: "/v1/responses",
			Handler: markerHandler{requested: invoked},
		},
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/responses", nil))

	if !*invoked {
		t.Fatalf("injected route was not reachable through the middleware chain")
	}
}

func TestRouterAdditionalRouteRejectsOtherMethods(t *testing.T) {
	invoked := new(bool)

	handler := server.NewRouter(
		testLogger(new(bytes.Buffer)),
		health.Live(),
		health.NewRegistry().Ready(),
		server.Route{Method: http.MethodPost, Pattern: "/v1/models", Handler: markerHandler{requested: invoked}},
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/models", nil))

	if *invoked {
		t.Fatalf("GET must not reach a POST-only route")
	}

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", recorder.Code)
	}
}
