package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/access"
)

func assertPipelineError(t *testing.T, recorder *httptest.ResponseRecorder, status int, wantType string) {
	t.Helper()

	if recorder.Code != status {
		t.Fatalf("status = %d, want %d", recorder.Code, status)
	}

	if got, want := recorder.Header().Get("Cache-Control"), "no-store"; got != want {
		t.Fatalf("Cache-Control = %q, want %q", got, want)
	}

	if got, want := recorder.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}

	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}

	if envelope.Error.Type != wantType {
		t.Fatalf("error type = %q, want %q", envelope.Error.Type, wantType)
	}

	if envelope.Error.Message == "" {
		t.Fatal("error message must not be empty")
	}
}

func TestPipelineErrorContract(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1024))

	t.Run("unauthorized", func(t *testing.T) {
		request := pipelineRequest(http.MethodGet, "/v1/models", "", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assertPipelineError(t, recorder, http.StatusUnauthorized, "unauthorized")
		if got, want := recorder.Header().Get("WWW-Authenticate"), `Bearer realm="ai-hub"`; got != want {
			t.Fatalf("WWW-Authenticate = %q, want %q", got, want)
		}

		if recorder.Header().Get("X-Request-Id") == "" {
			t.Fatal("error response must carry the correlation id")
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		request := pipelineRequest(http.MethodGet, "/v1/models", "", authHeaders())
		request.Host = "evil.test"
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assertPipelineError(t, recorder, http.StatusForbidden, "forbidden")
		if strings.Contains(recorder.Body.String(), "evil.test") {
			t.Fatal("response must not echo the rejected host")
		}
	})

	t.Run("payload too large", func(t *testing.T) {
		headers := authHeaders()
		headers["Content-Type"] = []string{"application/json"}
		request := pipelineRequest(http.MethodPost, "/v1/responses", strings.Repeat("a", 1025), headers)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assertPipelineError(t, recorder, http.StatusRequestEntityTooLarge, "payload_too_large")
	})

	t.Run("unsupported media type", func(t *testing.T) {
		request := pipelineRequest(http.MethodPost, "/v1/responses", `{"model":"x"}`, authHeaders())
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assertPipelineError(t, recorder, http.StatusUnsupportedMediaType, "unsupported_media_type")
	})

	t.Run("duplicate content type", func(t *testing.T) {
		headers := authHeaders()
		headers["Content-Type"] = []string{"application/json", "application/json"}
		request := pipelineRequest(http.MethodPost, "/v1/responses", `{"model":"x"}`, headers)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assertPipelineError(t, recorder, http.StatusBadRequest, "invalid_request")
	})
}

func TestPipelineDuplicateAuthorization(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1<<20))

	headers := map[string][]string{
		"Authorization": {authorization(), authorization()},
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, pipelineRequest(http.MethodGet, "/v1/models", "", headers))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called with duplicate authorization")
	}
}

func TestPipelineDuplicateOrigin(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, pipelinePolicy(t, []string{"https://app.example.com"}), probe, access.ResponsesAdmission(1<<20))

	headers := authHeaders()
	headers["Origin"] = []string{"https://app.example.com", "https://app.example.com"}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, pipelineRequest(http.MethodGet, "/v1/models", "", headers))

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestPipelineHostPrecedesAuthentication(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1<<20))

	request := pipelineRequest(http.MethodGet, "/v1/models", "", nil)
	request.Host = "evil.test"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestPipelineDoesNotLogSecrets(t *testing.T) {
	const (
		bodyMarker  = "body-marker-secret"
		queryMarker = "query-marker-secret"
	)

	logBuffer := &bytes.Buffer{}
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, logBuffer, testPolicy(t), probe, access.ResponsesAdmission(1<<20))

	headers := authHeaders()
	headers["Content-Type"] = []string{"application/json"}
	request := pipelineRequest(http.MethodPost, "/v1/responses?token="+queryMarker, `{"secret":"`+bodyMarker+`"}`, headers)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	logs := logBuffer.String()
	for _, secret := range []string{gatewayTestKey, bodyMarker, queryMarker, "Bearer"} {
		if strings.Contains(logs, secret) {
			t.Fatalf("logs must not contain %q: %q", secret, logs)
		}
	}
}

func TestPipelinePreservesInboundCorrelationIDOnError(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1<<20))

	request := pipelineRequest(http.MethodGet, "/v1/models", "", nil)
	request.Header.Set("X-Request-Id", "client-correlation-0001")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got, want := recorder.Header().Get("X-Request-Id"), "client-correlation-0001"; got != want {
		t.Fatalf("X-Request-Id = %q, want %q", got, want)
	}
}
