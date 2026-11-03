package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/observability"
)

func TestResponsesRoutingObservabilitySuccessPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer upstream.Close()

	logger, logBuffer := newTestLogger()
	handler := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "routed-model")

	body := `{"input":[{"type":"input_text","text":"secret prompt must not appear in logs"}]}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}

	logged := logBuffer.String()
	for _, required := range []string{
		"routing decision",
		"intelligent_routing=true",
		"reached_execution=true",
		"selection_origin=compatibility_default",
		"provider_id=opencode-go",
		"model_id=routed-model",
		"execution started",
		"execution finished",
	} {
		if !logBuffer.contains(required) {
			t.Fatalf("log missing %q in %q", required, logged)
		}
	}
	if strings.Contains(logged, "secret prompt") {
		t.Fatal("routing log leaked request content")
	}
}

func TestResponsesRoutingObservabilityCorrelatesRequestID(t *testing.T) {
	const requestID = "a1b2c3d4e5f67890"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer upstream.Close()

	logger, logBuffer := newTestLogger()
	handler := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "routed-model")

	body := `{"input":[{"type":"input_text","text":"hello"}]}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	request = request.WithContext(observability.WithCorrelationID(request.Context(), requestID))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}

	logged := logBuffer.String()
	marker := "request_id=" + requestID
	for _, event := range []string{"routing decision", "execution started", "execution finished"} {
		found := false
		for _, line := range strings.Split(logged, "\n") {
			if strings.Contains(line, event) && strings.Contains(line, marker) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no %q line with %q in log", event, marker)
		}
	}

	if !logBuffer.contains("reached_execution=true") {
		t.Fatalf("log = %q", logged)
	}
}

func TestResponsesRoutingObservabilityRejectionPath(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()

	logger, logBuffer := newTestLogger()
	handler := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "model-a", "model-b")

	body := `{"input":[{"type":"input_text","text":"hidden"}]}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d", response.Code)
	}

	logged := logBuffer.String()
	if !logBuffer.contains("routing decision") {
		t.Fatalf("log = %q", logged)
	}
	if !logBuffer.contains("rejection=routing_ambiguous") {
		t.Fatalf("log = %q", logged)
	}
	if !logBuffer.contains("reached_execution=false") {
		t.Fatalf("log = %q", logged)
	}
	if strings.Contains(logged, "execution finished") {
		t.Fatal("rejected route must not finish execution")
	}
	if strings.Contains(logged, "hidden") {
		t.Fatal("rejection log leaked request content")
	}
}

func TestResponsesRoutingObservabilityStructuralRejection(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()

	logger, logBuffer := newTestLogger()
	handler := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "only-model")

	body := `{"model":"","input":"x"}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}

	logged := logBuffer.String()
	if !logBuffer.contains("structural_rejection=invalid_request") {
		t.Fatalf("log = %q", logged)
	}
	if !logBuffer.contains("reached_execution=false") {
		t.Fatalf("log = %q", logged)
	}
}
