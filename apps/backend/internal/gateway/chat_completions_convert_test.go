package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChatRequestToResponsesSimpleUserMessage(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"model":"aihub/auto","messages":[{"role":"user","content":"oi"}]}`)
	out, model, stream, err := chatRequestToResponses(raw)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if model != "aihub/auto" || stream {
		t.Fatalf("model=%q stream=%v", model, stream)
	}
	if !strings.Contains(string(out), `"input":"oi"`) {
		t.Fatalf("payload = %s", string(out))
	}
}

func TestChatRequestToResponsesRejectsMissingModel(t *testing.T) {
	t.Parallel()

	_, _, _, err := chatRequestToResponses([]byte(`{"messages":[{"role":"user","content":"oi"}]}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatRequestNormalizesOpenAIToolSpecs(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"model":"aihub/auto","messages":[{"role":"user","content":"oi"}],"tools":[{"type":"function","function":{"name":"grep","description":"search","parameters":{"type":"object"}}}]}`)
	out, _, _, err := chatRequestToResponses(raw)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(string(out), `"name":"grep"`) || strings.Contains(string(out), `"function":{`) {
		t.Fatalf("payload = %s", string(out))
	}
}

func TestChatRequestSystemStringContentUsesInputText(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"model":"aihub/auto","messages":[
		{"role":"system","content":"You are a reviewer"},
		{"role":"user","content":"review this"}
	]}`)
	out, _, _, err := chatRequestToResponses(raw)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(string(out), `"type":"output_text"`) {
		t.Fatalf("system/user must not produce output_text in request input: %s", string(out))
	}
	if !strings.Contains(string(out), `"type":"input_text"`) {
		t.Fatalf("payload = %s", string(out))
	}
}

func TestChatRequestNormalizesCursorTextParts(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"model":"aihub/auto","messages":[
		{"role":"user","content":"oi"},
		{"role":"assistant","content":[{"type":"text","text":"ola"}]},
		{"role":"user","content":[{"type":"text","text":"de novo"}]}
	]}`)
	out, _, _, err := chatRequestToResponses(raw)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(string(out), `"type":"text"`) {
		t.Fatalf("payload still contains chat text parts: %s", string(out))
	}
	if !strings.Contains(string(out), `"type":"input_text"`) || !strings.Contains(string(out), `"type":"output_text"`) {
		t.Fatalf("payload = %s", string(out))
	}
}

func TestResponsesBodyToChatCompletion(t *testing.T) {
	t.Parallel()

	body := []byte(`{"id":"resp_1","model":"cursor/composer-2.5","output":[{"type":"message","content":[{"type":"output_text","text":"ola"}]}]}`)
	out, err := responsesBodyToChatCompletion(body, "aihub/auto")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	var chat map[string]any
	if err := json.Unmarshal(out, &chat); err != nil {
		t.Fatalf("json: %v", err)
	}
	if chat["object"] != "chat.completion" || chat["model"] != "aihub/auto" {
		t.Fatalf("chat payload = %v", chat)
	}
}
