package gateway_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/providers"
)

const (
	openCodeTestKey   = "sk-test-opencode-not-real"
	openRouterTestKey = "sk-test-openrouter-not-real"
)

func TestResponsesSelectsProviderByIDAndDefaultsToOpenCode(t *testing.T) {
	var openCodeHits, openRouterHits int
	openCodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openCodeHits++
		if r.URL.Path != "/openai/v1/responses" || r.Header.Get("Authorization") != "Bearer "+openCodeTestKey {
			t.Errorf("OpenCode request = %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"opencode-response"}`)
	}))
	defer openCodeServer.Close()
	openRouterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openRouterHits++
		if r.URL.Path != "/api/v1/responses" || r.Header.Get("Authorization") != "Bearer "+openRouterTestKey {
			t.Errorf("OpenRouter request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Opencode-Session") != "" {
			t.Error("OpenCode session header was forwarded to OpenRouter")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"openrouter-response"}`)
	}))
	defer openRouterServer.Close()
	handler, _, openRouter := newRoutedResponsesHandler(t, openCodeServer.URL, openRouterServer.URL)
	payload := `{"model":"anthropic/claude-sonnet-4.5","input":"hello"}`

	defaultResponse := serveResponses(handler, "", `{"input":[{"type":"input_text","text":"hello"}]}`)
	if defaultResponse.Code != http.StatusOK || !strings.Contains(defaultResponse.Body.String(), "opencode-response") {
		t.Fatalf("default response = %d %s", defaultResponse.Code, defaultResponse.Body.String())
	}
	routedResponse := serveResponses(handler, providers.OpenRouterID, payload)
	if routedResponse.Code != http.StatusOK || !strings.Contains(routedResponse.Body.String(), "openrouter-response") {
		t.Fatalf("routed response = %d %s", routedResponse.Code, routedResponse.Body.String())
	}
	if openCodeHits != 1 || openRouterHits != 1 || !openRouter.Enabled() {
		t.Fatalf("hits/enabled = %d/%d/%t", openCodeHits, openRouterHits, openRouter.Enabled())
	}
}

func TestResponsesRejectsUnknownDisabledAndDuplicateProviderSelectors(t *testing.T) {
	openCodeServer := httptest.NewServer(http.NotFoundHandler())
	defer openCodeServer.Close()
	openRouterServer := httptest.NewServer(http.NotFoundHandler())
	defer openRouterServer.Close()
	handler, _, openRouter := newRoutedResponsesHandler(t, openCodeServer.URL, openRouterServer.URL)

	if got := serveResponses(handler, "unregistered", `{}`).Code; got != http.StatusNotFound {
		t.Fatalf("unknown provider status = %d, want 404", got)
	}
	if err := openRouter.Reconfigure(providers.Settings{BaseURL: openRouterServer.URL + "/api/v1", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if got := serveResponses(handler, providers.OpenRouterID, `{}`).Code; got != http.StatusServiceUnavailable {
		t.Fatalf("credentialless provider status = %d, want 503", got)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`))
	request.Header.Add("X-AIHub-Provider", providers.OpenRouterID)
	request.Header.Add("X-AIHub-Provider", providers.OpenRouterID)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("duplicate provider selector status = %d, want 400", response.Code)
	}
}

func TestResponsesSanitizesOpenRouterHTTPErrorBodyWithoutFallback(t *testing.T) {
	var openCodeHits int
	openCode := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		openCodeHits++
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"id":"fallback-response"}`)
	}))
	defer openCode.Close()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(writer, `{"error":{"message":"invalid credential `+openRouterTestKey+`"}}`)
	}))
	defer upstream.Close()

	handler, _, _ := newRoutedResponsesHandler(t, openCode.URL, upstream.URL)
	response := serveResponses(handler, providers.OpenRouterID, `{}`)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if openCodeHits != 0 {
		t.Fatalf("OpenCode Go received %d requests after OpenRouter failed; want no fallback", openCodeHits)
	}
	if strings.Contains(response.Body.String(), openRouterTestKey) {
		t.Fatal("Responses error body exposed the OpenRouter credential")
	}
	if !strings.Contains(response.Body.String(), "upstream_error") {
		t.Fatal("Responses error body omitted the sanitized upstream error type")
	}
}

func serveResponses(handler http.Handler, providerID, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if providerID != "" {
		request.Header.Set("X-AIHub-Provider", providerID)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
