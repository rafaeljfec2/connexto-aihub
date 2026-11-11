package gateway_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func TestModelsHandlerReturnsUpstreamCatalog(t *testing.T) {
	const catalog = `{"object":"list","data":[{"id":"deepseek-v4.1-flash","name":"DeepSeek V4.1 Flash","api":"chat-completions"}]}`

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("upstream saw %s %s, want GET /v1/models", r.Method, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(catalog))
	}))
	defer upstream.Close()

	logger, _ := newTestLogger()
	handler := gateway.NewModelsHandler(newEnabledClient(t, upstream.URL), logger)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, newModelRequest(t))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if got, want := recorder.Body.String(), catalog; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestModelsHandlerRejectsProviderWithoutCredential(t *testing.T) {
	disabled, err := providers.NewClient(emptyProviderConfig(), nil)
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}

	logger, _ := newTestLogger()
	handler := gateway.NewModelsHandler(disabled, logger)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, newModelRequest(t))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}

	if strings.Contains(recorder.Body.String(), providerCredential) {
		t.Fatalf("body leaks the credential")
	}
}

func TestModelsHandlerAnswersSafe502WhenUpstreamUnreachable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	dead := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	logger, _ := newTestLogger()
	handler := gateway.NewModelsHandler(newEnabledClient(t, "http://"+dead), logger)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, newModelRequest(t))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", recorder.Code)
	}

	body := recorder.Body.String()
	if strings.Contains(body, providerCredential) || strings.Contains(body, dead) {
		t.Fatalf("502 body must be safe, got %q", body)
	}
}

func TestModelsHandlerPreservesUpstreamErrorStatus(t *testing.T) {
	const errorBody = `{"error":{"type":"server_error","message":"Upstream request failed: Model is unavailable."}}`

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(errorBody))
	}))
	defer upstream.Close()

	logger, _ := newTestLogger()
	handler := gateway.NewModelsHandler(newEnabledClient(t, upstream.URL), logger)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, newModelRequest(t))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want the upstream 400 preserved", recorder.Code)
	}

	if got := recorder.Body.String(); got != errorBody {
		t.Fatalf("body = %q, want the upstream envelope", got)
	}
}

func newModelRequest(t *testing.T) *http.Request {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, "/v1/models", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	return request
}
