package providers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/providers"
)

func TestOpenRouterResponsesSurfacesRedactedUpstreamMessage(t *testing.T) {
	const wantMessage = "Missing required parameter: model"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"code":"invalid_prompt","message":"`+wantMessage+`"}}`)
	}))
	defer server.Close()

	client := newOpenRouterTestClient(t, server.URL+"/api/v1", server.Client(), openRouterTestCredential)
	response, err := client.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenRouterID,
	}, []byte(`{"model":"anthropic/claude-haiku-4.5","input":"hello"}`), "application/json", "")
	if err != nil {
		t.Fatalf("OpenTargetResponses() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.StatusCode)
	}
	detail := response.Header.Get(providers.InternalUpstreamErrorDetailHeader)
	if detail == "" {
		t.Fatal("expected internal upstream error detail header")
	}
	if !strings.Contains(detail, wantMessage) || !strings.Contains(detail, "invalid_prompt") {
		t.Fatalf("internal detail = %q", detail)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if envelope.Error.Type != "upstream_error" {
		t.Fatalf("error type = %q", envelope.Error.Type)
	}
	if !strings.Contains(envelope.Error.Message, wantMessage) {
		t.Fatalf("client message = %q, want substring %q", envelope.Error.Message, wantMessage)
	}
	if strings.Contains(string(body), openRouterTestCredential) {
		t.Fatal("credential leaked in sanitized body")
	}
}

func TestOpenRouterResponsesRedactsCredentialSnippetsInUpstreamMessage(t *testing.T) {
	leaked := `Authorization Bearer ` + openRouterTestCredential + ` rejected`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"`+leaked+`"}}`)
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

	detail := response.Header.Get(providers.InternalUpstreamErrorDetailHeader)
	if strings.Contains(detail, openRouterTestCredential) || strings.Contains(detail, "sk-or-test") {
		t.Fatalf("detail leaked credential: %q", detail)
	}
	body, _ := io.ReadAll(response.Body)
	if strings.Contains(string(body), openRouterTestCredential) {
		t.Fatalf("body leaked credential: %s", body)
	}
}

func TestOpenRouterResponsesFallbackMessageWhenUpstreamBodyUnparseable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `<html>upstream outage</html>`)
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

	body, _ := io.ReadAll(response.Body)
	if !strings.Contains(string(body), "upstream outage") {
		t.Fatalf("body = %s, want redacted plaintext snippet", body)
	}
	if strings.Contains(string(body), "<html>") {
		t.Fatalf("body should not expose raw html tags: %s", body)
	}
}

func TestOpenRouterResponsesIncludesProviderCodeInDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"provider rejected","metadata":{"provider_code":"invalid_model"}}}`)
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

	detail := response.Header.Get(providers.InternalUpstreamErrorDetailHeader)
	if !strings.Contains(detail, "provider rejected") || !strings.Contains(detail, "provider_code=invalid_model") {
		t.Fatalf("detail = %q", detail)
	}
}
