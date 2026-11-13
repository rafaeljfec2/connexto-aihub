package gateway_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/gateway"
)

func TestChatCompletionsHandlerRequiresModel(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	logger, _ := newTestLogger()
	responses := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "selected-model")
	handler := gateway.NewChatCompletionsHandler(responses)

	body := `{"messages":[{"role":"user","content":"oi"}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "model is required") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestChatCompletionsHandlerConvertsNonStreamingResponse(t *testing.T) {
	var captured upstreamRequest
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = upstreamRequest{Body: string(body)}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"resp_x","model":"selected-model","output":[{"type":"message","content":[{"type":"output_text","text":"pong"}]}]}`)
	}))
	defer upstream.Close()

	logger, _ := newTestLogger()
	responses := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "selected-model")
	handler := gateway.NewChatCompletionsHandler(responses)

	body := `{"model":"selected-model","messages":[{"role":"user","content":"ping"}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(captured.Body, `"input":"ping"`) {
		t.Fatalf("upstream body = %s", captured.Body)
	}

	var chat map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &chat); err != nil {
		t.Fatalf("json: %v", err)
	}
	if chat["object"] != "chat.completion" || chat["model"] != "selected-model" {
		t.Fatalf("chat = %v", chat)
	}
}

func TestChatCompletionsHandlerConversionFailureReturnsBadGateway(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `not-json{{{`)
	}))
	defer upstream.Close()

	logger, _ := newTestLogger()
	responses := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "selected-model")
	handler := gateway.NewChatCompletionsHandler(responses)

	body := `{"model":"selected-model","messages":[{"role":"user","content":"ping"}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "not-json") {
		t.Fatalf("must not leak raw Responses body: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "could not be converted") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestChatCompletionsRoutedVirtualAutoPreservesClientModel(t *testing.T) {
	openCode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"resp_auto","model":"composer-2.5","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`)
	}))
	defer openCode.Close()
	openRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"or","model":"haiku","output":[]}`)
	}))
	defer openRouter.Close()

	routed, _, _ := newRoutedResponsesHandler(t, openCode.URL, openRouter.URL)
	handler := gateway.NewChatCompletionsHandler(routed)

	body := `{"model":"aihub/auto","messages":[{"role":"user","content":"hi"}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	var chat map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &chat); err != nil {
		t.Fatalf("json: %v", err)
	}
	if chat["model"] != "aihub/auto" {
		t.Fatalf("model = %v", chat["model"])
	}
}
