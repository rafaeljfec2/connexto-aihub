package providers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/providers"
)

const openRouterTestCredential = "sk-or-test-only-do-not-use"

func TestOpenRouterResponsesPreservesBodyAndUsesConfiguredEndpoint(t *testing.T) {
	payload := `{"model":"anthropic/claude-sonnet-4.5","input":"hello","stream":true}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/responses" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+openRouterTestCredential {
			t.Errorf("authorization header mismatch")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q", got)
		}
		if got := r.Header.Get("X-Opencode-Session"); got != "" {
			t.Errorf("OpenCode session header leaked to OpenRouter: %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if string(body) != payload {
			t.Errorf("body = %q, want exact payload %q", body, payload)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := newOpenRouterTestClient(t, server.URL+"/api/v1", server.Client(), openRouterTestCredential)
	response, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenRouterID,
		ModelID:    "anthropic/claude-sonnet-4.5",
	}, []byte(payload), "application/json", "session-1")
	if err != nil {
		t.Fatalf("OpenTargetResponses() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}

func TestOpenRouterFetchModelDescriptorsPreservesProviderModelIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/models" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+openRouterTestCredential {
			t.Errorf("authorization header mismatch")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"id":"anthropic/claude-sonnet-4.5","name":"Claude Sonnet","architecture":{"input_modalities":["text","image"],"output_modalities":["text"]}}]}`)
	}))
	defer server.Close()

	client := newOpenRouterTestClient(t, server.URL+"/api/v1", server.Client(), openRouterTestCredential)
	descriptors, err := client.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}
	if len(descriptors) != 1 || descriptors[0].ID != "anthropic/claude-sonnet-4.5" {
		t.Fatalf("descriptors = %+v", descriptors)
	}
	if descriptors[0].Name != "Claude Sonnet" || !strings.Contains(string(descriptors[0].Raw), "input_modalities") {
		t.Fatalf("descriptor metadata = %+v", descriptors[0])
	}
	if descriptors[0].ToolsSupported != nil || descriptors[0].ReasoningSupported != nil {
		t.Fatalf("unreported capabilities were asserted: %+v", descriptors[0])
	}
	if !descriptors[0].InputModalitiesPresent {
		t.Fatal("InputModalitiesPresent = false")
	}
	if !containsModality(descriptors[0].InputModalities, "image") || !containsModality(descriptors[0].InputModalities, "image/png") {
		t.Fatalf("InputModalities = %v, want bare image and expanded image/png", descriptors[0].InputModalities)
	}
}

func containsModality(modalities []string, want string) bool {
	for _, modality := range modalities {
		if modality == want {
			return true
		}
	}
	return false
}

func TestOpenRouterFetchModelDescriptorsRejectsInvalidPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":[{"name":"missing id"}]}`)
	}))
	defer server.Close()

	client := newOpenRouterTestClient(t, server.URL+"/api/v1", server.Client(), openRouterTestCredential)
	if _, err := client.FetchModelDescriptors(context.Background()); err == nil {
		t.Fatal("FetchModelDescriptors() accepted a model without an id")
	}
}

func TestOpenRouterResponsesPreservesUpstreamHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"code":"rate_limit_exceeded"}}`)
	}))
	defer server.Close()

	client := newOpenRouterTestClient(t, server.URL+"/api/v1", server.Client(), openRouterTestCredential)
	response, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenRouterID,
	}, []byte(`{"model":"openai/model-a","input":"hello"}`), "application/json", "")
	if err != nil {
		t.Fatalf("OpenTargetResponses() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", response.StatusCode)
	}
}

func TestOpenRouterResponsesMapsHeaderTimeoutWithoutCredential(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)

	transport := &http.Transport{ResponseHeaderTimeout: 20 * time.Millisecond}
	client := newOpenRouterTestClient(t, server.URL+"/api/v1", &http.Client{Transport: transport}, openRouterTestCredential)
	_, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenRouterID,
	}, []byte(`{"model":"openai/model-a","input":"hello"}`), "application/json", "")
	var timeout providers.ResponseHeaderTimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("error = %v, want response header timeout", err)
	}
	if strings.Contains(err.Error(), openRouterTestCredential) {
		t.Fatal("timeout error exposed the credential")
	}
}

func TestOpenRouterFetchModelDescriptorsExtractsCapabilitiesAndLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"data": [
				{
					"id": "openai/gpt-5.2",
					"name": "OpenAI: GPT-5.2",
					"context_length": 400000,
					"top_provider": {
						"max_completion_tokens": 128000,
						"context_length": 400000
					},
					"supported_parameters": [
						"tools",
						"reasoning",
						"structured_outputs"
					]
				}
			]
		}`)
	}))
	defer server.Close()

	client := newOpenRouterTestClient(t, server.URL+"/api/v1", server.Client(), openRouterTestCredential)
	descriptors, err := client.FetchModelDescriptors(context.Background())
	if err != nil {
		t.Fatalf("FetchModelDescriptors() error = %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("len(descriptors) = %d, want 1", len(descriptors))
	}
	d := descriptors[0]
	if d.LimitContext == nil || *d.LimitContext != 400000 {
		t.Fatalf("LimitContext = %v, want 400000", d.LimitContext)
	}
	if d.LimitOutput == nil || *d.LimitOutput != 128000 {
		t.Fatalf("LimitOutput = %v, want 128000", d.LimitOutput)
	}
	if d.ToolsSupported == nil || !*d.ToolsSupported {
		t.Fatalf("ToolsSupported = %v, want true", d.ToolsSupported)
	}
	if d.ReasoningSupported == nil || !*d.ReasoningSupported {
		t.Fatalf("ReasoningSupported = %v, want true", d.ReasoningSupported)
	}
	if d.StructuredOutput == nil || !*d.StructuredOutput {
		t.Fatalf("StructuredOutput = %v, want true", d.StructuredOutput)
	}
}

func newOpenRouterTestClient(t *testing.T, endpoint string, client *http.Client, credential string) *providers.OpenRouterClient {
	t.Helper()
	adapter, err := providers.NewOpenRouterClient(providers.Settings{
		BaseURL: endpoint,
		APIKey:  credential,
		Enabled: true,
	}, client)
	if err != nil {
		t.Fatalf("NewOpenRouterClient() error = %v", err)
	}
	return adapter
}
