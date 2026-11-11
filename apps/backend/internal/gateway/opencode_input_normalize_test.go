package gateway

import (
	"encoding/json"
	"testing"

	"aihub/internal/providers"
)

func TestNormalizeOpenCodeGoResponsesInputWrapsFlatParts(t *testing.T) {
	in := []byte(`{"model":"gpt-6-luna","input":[{"type":"input_text","text":"hi"},{"type":"input_image","image_url":"data:image/png;base64,abc"}]}`)
	out, err := normalizeOpenCodeGoResponsesInput(in)
	if err != nil {
		t.Fatalf("normalize error = %v", err)
	}

	var envelope struct {
		Input []struct {
			Type    string           `json:"type"`
			Role    string           `json:"role"`
			Content []map[string]any `json:"content"`
		} `json:"input"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(envelope.Input) != 1 {
		t.Fatalf("input len = %d, want 1 message", len(envelope.Input))
	}
	if envelope.Input[0].Type != "message" || envelope.Input[0].Role != "user" {
		t.Fatalf("message = %+v", envelope.Input[0])
	}
	if len(envelope.Input[0].Content) != 2 {
		t.Fatalf("content len = %d", len(envelope.Input[0].Content))
	}
}

func TestNormalizeOpenCodeGoResponsesInputAddsMessageTypeToRoleOnly(t *testing.T) {
	in := []byte(`{"model":"gpt-6-luna","input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}]}`)
	out, err := normalizeOpenCodeGoResponsesInput(in)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Input []map[string]any `json:"input"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Input[0]["type"] != "message" {
		t.Fatalf("type = %v, want message", envelope.Input[0]["type"])
	}
}

func TestAdaptResponsesPayloadForProviderSkipsOtherProviders(t *testing.T) {
	body := []byte(`{"input":[{"type":"input_text","text":"x"}]}`)
	out, err := adaptResponsesPayloadForProvider(providers.CursorID, body)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(body) {
		t.Fatalf("cursor payload changed: %s", out)
	}
}

func TestAdaptResponsesPayloadForProviderOpenCodeGo(t *testing.T) {
	body := []byte(`{"input":[{"type":"input_text","text":"x"}]}`)
	out, err := adaptResponsesPayloadForProvider(providers.OpenCodeGoID, body)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) == string(body) {
		t.Fatal("expected opencode normalization")
	}
}
