package gateway_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/providers"
)

func TestResponsesRoutingVerticalSliceInjectsSelectedModel(t *testing.T) {
	var capture upstreamRequest
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capture = upstreamRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"routed"}`)
	}))
	defer upstream.Close()

	logger, _ := newTestLogger()
	handler := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "selected-model")

	body := `{"input":[{"type":"input_text","text":"hello"}]}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}

	var forwarded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(capture.Body), &forwarded); err != nil {
		t.Fatalf("upstream body json: %v", err)
	}

	var modelID string
	if err := json.Unmarshal(forwarded["model"], &modelID); err != nil {
		t.Fatalf("upstream model field: %v body=%s", err, capture.Body)
	}

	if modelID != "selected-model" {
		t.Fatalf("upstream model = %q, want selected-model", modelID)
	}
}

func TestResponsesRoutingRejectsAmbiguousSelection(t *testing.T) {
	var upstreamHits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits++
		http.NotFound(w, r)
	}))
	defer upstream.Close()

	logger, _ := newTestLogger()
	handler := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "model-a", "model-b")

	body := `{"input":[{"type":"input_text","text":"hello"}]}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.Code)
	}

	if !strings.Contains(response.Body.String(), "routing_ambiguous") {
		t.Fatalf("body = %s", response.Body.String())
	}

	if upstreamHits != 0 {
		t.Fatalf("upstream hits = %d, want 0 on routing rejection", upstreamHits)
	}
}

func TestResponsesRoutingExplicitOpenRouterModelOnly(t *testing.T) {
	var providerHit string
	openCode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		providerHit = providers.OpenCodeGoID
		w.WriteHeader(http.StatusOK)
	}))
	defer openCode.Close()
	openRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		providerHit = providers.OpenRouterID
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	defer openRouter.Close()

	handler, _, _ := newRoutedResponsesHandler(t, openCode.URL, openRouter.URL)
	payload := `{"model":"anthropic/claude-haiku-4.5","input":[{"type":"input_text","text":"hi"}]}`
	response := serveResponses(handler, "", payload)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if providerHit != providers.OpenRouterID {
		t.Fatalf("provider hit = %q, want openrouter", providerHit)
	}
}

func TestResponsesRoutingVirtualAutoInjectsHeadersAndSubstitutesPayload(t *testing.T) {
	var capturedBody string
	openCode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"virtual-resp"}`)
	}))
	defer openCode.Close()
	openRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"openrouter-resp"}`)
	}))
	defer openRouter.Close()

	handler, _, _ := newRoutedResponsesHandler(t, openCode.URL, openRouter.URL)
	payload := `{"model":"aihub/auto","input":[{"type":"input_text","text":"hi"}]}`
	response := serveResponses(handler, "", payload)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}

	// Verify pre-commit transparency headers
	resolvedModel := response.Header().Get("X-AIHub-Resolved-Model")
	if resolvedModel == "" {
		t.Errorf("expected X-AIHub-Resolved-Model header, got empty")
	}
	resolvedProvider := response.Header().Get("X-AIHub-Resolved-Provider")
	if resolvedProvider == "" {
		t.Errorf("expected X-AIHub-Resolved-Provider header, got empty")
	}
	profile := response.Header().Get("X-AIHub-Routing-Profile")
	if profile != "balanced" {
		t.Errorf("expected X-AIHub-Routing-Profile = 'balanced', got %q", profile)
	}
	reason := response.Header().Get("X-AIHub-Routing-Reason")
	allowedReasons := map[string]bool{
		"awrr_pool_select":                 true,
		"reasoning_preferred":              true,
		"reasoning_preference_unmatched":   true,
		"efficiency_fast_path":             true,
	}
	if !allowedReasons[reason] {
		t.Errorf("expected controlled X-AIHub-Routing-Reason, got %q", reason)
	}

	// Verify upstream did NOT receive the virtual model string
	if strings.Contains(capturedBody, "aihub/auto") {
		t.Fatalf("upstream payload still contains 'aihub/auto': %s", capturedBody)
	}

	// Verify upstream received the resolved model
	if !strings.Contains(capturedBody, resolvedModel) {
		t.Fatalf("upstream payload does not contain resolved model %q: %s", resolvedModel, capturedBody)
	}
}
