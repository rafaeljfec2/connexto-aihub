package gateway

import (
	"strings"
	"testing"

	"aihub/internal/providers"
)

func TestOpenRouterResponsesInputStripsRedactedThinking(t *testing.T) {
	in := []byte(`{
		"model":"anthropic/claude-haiku-4.5",
		"stream":true,
		"include":["reasoning.encrypted_content"],
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]},
			{"type":"message","role":"assistant","content":[
				{"type":"redacted_thinking","data":"opaque"},
				{"type":"output_text","text":"hello back"}
			]}
		]
	}`)

	out, err := OpenRouterResponsesPayloadForUpstream(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "redacted_thinking") {
		t.Fatalf("output still contains redacted_thinking: %s", text)
	}
	if !strings.Contains(text, "output_text") || !strings.Contains(text, "hello back") {
		t.Fatalf("assistant text removed: %s", text)
	}
	if strings.Contains(text, "reasoning.encrypted_content") {
		t.Fatalf("include not stripped: %s", text)
	}
}

func TestOpenRouterResponsesWrapsFlatInputAndFunctionTools(t *testing.T) {
	in := []byte(`{
		"model":"anthropic/claude-haiku-4.5",
		"instructions":"e2e-stay-intact",
		"tool_choice":"auto",
		"tools":[{"type":"function","name":"probe_tool","parameters":{"type":"object","properties":{}}}],
		"input":[{"type":"input_text","text":"call probe_tool"}],
		"stream":false
	}`)
	out, err := OpenRouterResponsesPayloadForUpstream(in)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "e2e-stay-intact") {
		t.Fatalf("instructions changed: %s", text)
	}
	if !strings.Contains(text, `"type":"message"`) {
		t.Fatalf("input not wrapped: %s", text)
	}
	if !strings.Contains(text, `"type":"custom"`) || !strings.Contains(text, `"name":"probe_tool"`) {
		t.Fatalf("function tools should map to custom for OpenRouter: %s", text)
	}
}

func TestAdaptResponsesPayloadForProviderOpenRouterOnly(t *testing.T) {
	in := []byte(`{"input":[{"type":"message","role":"assistant","content":[{"type":"redacted_thinking"}]}]}`)
	openRouterOut, err := adaptResponsesPayloadForProvider(providers.OpenRouterID, in)
	if err != nil {
		t.Fatalf("openrouter: %v", err)
	}
	if strings.Contains(string(openRouterOut), "redacted_thinking") {
		t.Fatal("openrouter should strip thinking block")
	}

	otherOut, err := adaptResponsesPayloadForProvider(providers.CursorID, in)
	if err != nil {
		t.Fatalf("cursor: %v", err)
	}
	if !strings.Contains(string(otherOut), "redacted_thinking") {
		t.Fatal("unrelated providers should leave payload unchanged")
	}
}
