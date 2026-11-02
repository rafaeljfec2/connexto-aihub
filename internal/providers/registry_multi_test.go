package providers_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"aihub/internal/providers"
)

type routedTestAdapter struct {
	settings providers.Settings
	calls    []providers.ResponsesTarget
	payloads [][]byte
}

func (a *routedTestAdapter) Reconfigure(settings providers.Settings) error {
	a.settings = settings
	return nil
}

func (a *routedTestAdapter) Enabled() bool {
	return a.settings.Enabled && a.settings.APIKey != ""
}

func (a *routedTestAdapter) OpenTargetResponses(_ context.Context, target providers.ResponsesTarget, payload []byte, _, _ string) (*http.Response, error) {
	a.calls = append(a.calls, target)
	a.payloads = append(a.payloads, append([]byte(nil), payload...))
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
}

func TestRegistryRoutesResponsesByProviderIDAndKeepsSettingsIsolated(t *testing.T) {
	registry := providers.NewRegistry()
	openCode := &routedTestAdapter{}
	openRouter := &routedTestAdapter{}
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.OpenRouterID, openRouter)
	if err := openCode.Reconfigure(providers.Settings{BaseURL: "https://opencode.example.test", APIKey: "test-opencode", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := openRouter.Reconfigure(providers.Settings{BaseURL: "https://openrouter.example.test", APIKey: "test-openrouter", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{"model":"anthropic/claude-sonnet-4.5","input":"hello"}`)
	response, err := registry.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenRouterID,
		ModelID:    "anthropic/claude-sonnet-4.5",
	}, payload, "application/json", "session-1")
	if err != nil {
		t.Fatalf("OpenTargetResponses() error = %v", err)
	}
	_ = response.Body.Close()
	if len(openRouter.calls) != 1 || openRouter.calls[0].ModelID != "anthropic/claude-sonnet-4.5" {
		t.Fatalf("OpenRouter calls = %+v", openRouter.calls)
	}
	if len(openCode.calls) != 0 || string(openRouter.payloads[0]) != string(payload) {
		t.Fatalf("provider isolation/body mismatch: OpenCode calls=%d OpenRouter body=%q", len(openCode.calls), openRouter.payloads[0])
	}
	if openCode.settings.APIKey != "test-opencode" || openRouter.settings.APIKey != "test-openrouter" {
		t.Fatal("provider settings were shared across registry bindings")
	}
}

func TestRegistryRejectsUnboundAndNonResponsesAdapters(t *testing.T) {
	registry := providers.NewRegistry()
	registry.Bind("catalog-only", &recordingAdapter{})
	target := providers.ResponsesTarget{ProviderID: "missing"}
	if _, err := registry.OpenTargetResponses(context.Background(), target, nil, "application/json", ""); err == nil {
		t.Fatal("unbound provider accepted Responses traffic")
	}
	target.ProviderID = "catalog-only"
	if _, err := registry.OpenTargetResponses(context.Background(), target, nil, "application/json", ""); err == nil {
		t.Fatal("adapter without Responses support accepted traffic")
	}
}

func TestRegistryRefusesDisabledOrCredentiallessProvider(t *testing.T) {
	registry := providers.NewRegistry()
	adapter := &routedTestAdapter{}
	registry.Bind(providers.OpenRouterID, adapter)
	if err := adapter.Reconfigure(providers.Settings{BaseURL: "https://openrouter.example.test", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	_, err := registry.OpenTargetResponses(context.Background(), providers.ResponsesTarget{
		ProviderID: providers.OpenRouterID,
	}, []byte(`{"model":"openai/model-a"}`), "application/json", "")
	if err == nil || len(adapter.calls) != 0 {
		t.Fatalf("credentialless provider call = err %v, calls %d", err, len(adapter.calls))
	}
}
