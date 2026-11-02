package gateway_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func TestModelsRouteSelectsProviderByIDAndDefaultsToOpenCode(t *testing.T) {
	var openCodeHits, openRouterHits int
	openCodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openCodeHits++
		if r.URL.Path != "/v1/models" {
			t.Errorf("OpenCode models path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"deepseek-v4.1-flash"}]}`)
	}))
	defer openCodeServer.Close()
	openRouterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openRouterHits++
		if r.URL.Path != "/api/v1/models" {
			t.Errorf("OpenRouter models path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"anthropic/claude-sonnet-4.5"}]}`)
	}))
	defer openRouterServer.Close()
	openCode, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: openCodeServer.URL, APIKey: openCodeTestKey}, nil)
	if err != nil {
		t.Fatal(err)
	}
	openRouter, err := providers.NewOpenRouterClient(providers.Settings{BaseURL: openRouterServer.URL + "/api/v1", APIKey: openRouterTestKey, Enabled: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.OpenRouterID, openRouter)
	handler := gateway.NewRegistryModelsHandler(registry, slog.New(slog.NewTextHandler(io.Discard, nil)))

	defaultResponse := serveModels(handler, "")
	if defaultResponse.Code != http.StatusOK || !strings.Contains(defaultResponse.Body.String(), "deepseek-v4.1-flash") {
		t.Fatalf("default models response = %d %s", defaultResponse.Code, defaultResponse.Body.String())
	}
	routedResponse := serveModels(handler, providers.OpenRouterID)
	if routedResponse.Code != http.StatusOK || !strings.Contains(routedResponse.Body.String(), "anthropic/claude-sonnet-4.5") {
		t.Fatalf("OpenRouter models response = %d %s", routedResponse.Code, routedResponse.Body.String())
	}
	if openCodeHits != 1 || openRouterHits != 1 {
		t.Fatalf("model requests = OpenCode %d, OpenRouter %d", openCodeHits, openRouterHits)
	}
}

func TestModelsRouteSanitizesOpenRouterHTTPErrorBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(writer, `{"error":{"message":"invalid credential `+openRouterTestKey+`"}}`)
	}))
	defer upstream.Close()
	adapter, err := providers.NewOpenRouterClient(providers.Settings{
		BaseURL: upstream.URL + "/api/v1", APIKey: openRouterTestKey, Enabled: true,
	}, upstream.Client())
	if err != nil {
		t.Fatal(err)
	}
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenRouterID, adapter)
	handler := gateway.NewRegistryModelsHandler(registry, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := serveModels(handler, providers.OpenRouterID)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if strings.Contains(response.Body.String(), openRouterTestKey) {
		t.Fatal("model-list error body exposed the OpenRouter credential")
	}
	if !strings.Contains(response.Body.String(), "upstream_error") {
		t.Fatal("model-list error body omitted the sanitized upstream error type")
	}
}

func serveModels(handler http.Handler, providerID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	if providerID != "" {
		request.Header.Set(gateway.ProviderHeader, providerID)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
