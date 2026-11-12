package providers

import (
	"strings"
	"testing"
)

func TestNormalizeOpenCodeGoResponsesPayloadStripsNativeReasoningFields(t *testing.T) {
	body := []byte(`{
		"model":"glm-5.3-flash",
		"stream":true,
		"include":["reasoning.encrypted_content"],
		"reasoning":{"effort":"medium","context":{"turns":1},"summary":"x"},
		"input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}]
	}`)
	out, err := NormalizeOpenCodeGoResponsesPayload(body)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "context") || strings.Contains(text, `"summary"`) {
		t.Fatalf("native reasoning fields remain: %s", text)
	}
	if !strings.Contains(text, `"effort":"medium"`) {
		t.Fatal("portable reasoning effort should remain")
	}
	if strings.Contains(text, "reasoning.encrypted_content") {
		t.Fatal("include should be stripped")
	}
}
