package providers

import (
	"strings"
	"testing"
)

func TestNormalizeOpenRouterResponsesPayloadDeepScrub(t *testing.T) {
	body := []byte(`{
		"model":"anthropic/claude-haiku-4.5",
		"stream":true,
		"include":["reasoning.encrypted_content","reasoning.summary"],
		"input":[
			{"role":"user","content":[{"type":"input_text","text":"hi"}]},
			{"role":"assistant","content":[{"type":"redacted_thinking","data":"opaque"},{"type":"output_text","text":"ok"}]},
			{"type":"redacted_thinking","data":"top-level"},
			{"role":"assistant","content":[{"redacted_thinking":{"data":"nested-key"}}]}
		]
	}`)
	out, err := NormalizeOpenRouterResponsesPayload(body)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "redacted_thinking") {
		t.Fatalf("thinking not stripped: %s", text)
	}
	if !strings.Contains(text, "output_text") {
		t.Fatal("assistant text should remain")
	}
	if strings.Contains(text, "reasoning.encrypted_content") {
		t.Fatal("include should be stripped")
	}
}
