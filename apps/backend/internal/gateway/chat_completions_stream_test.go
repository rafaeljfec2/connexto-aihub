package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsResponsesStreamErrorIgnoresNullErrorField(t *testing.T) {
	t.Parallel()

	data := []byte(`{"type":"response.created","response":{"error":null,"status":"in_progress"}}`)
	if isResponsesStreamError("response.created", data) {
		t.Fatal("response.created with error:null must not trigger passthrough")
	}
}

func TestIsResponsesStreamErrorDetectsExplicitErrorEvent(t *testing.T) {
	t.Parallel()

	data := []byte(`{"type":"error","error":{"message":"boom"}}`)
	if !isResponsesStreamError("error", data) {
		t.Fatal("expected explicit stream error")
	}
}

func TestChatStreamTranslatorFunctionCallChunk(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	tr := newChatStreamTranslator(rec, "test-model", "chatcmpl_test")

	block := []byte(
		"event: response.output_item.done\n" +
			`data: {"type":"response.output_item.done","output_index":0,"item":{` +
			`"type":"function_call","status":"completed","call_id":"call_abc","name":"Shell","arguments":"{\"command\":\"git status\"}"}}`,
	)
	if err := tr.emitTranslatedBlock(block); err != nil {
		t.Fatal(err)
	}
	completed := []byte(
		"event: response.completed\n" +
			`data: {"type":"response.completed","response":{"output":[{"type":"function_call","call_id":"call_abc","name":"Shell","arguments":"{\"command\":\"git status\"}"}]}}`,
	)
	if err := tr.emitTranslatedBlock(completed); err != nil {
		t.Fatal(err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"tool_calls"`) {
		t.Fatalf("expected tool_calls chunk, got %q", body)
	}
	if !strings.Contains(body, `"finish_reason":"tool_calls"`) {
		t.Fatalf("expected finish_reason tool_calls, got %q", body)
	}
	if strings.Count(body, "[DONE]") != 1 {
		t.Fatalf("expected single DONE, got %q", body)
	}
	if strings.Count(body, "call_abc") != 1 {
		t.Fatalf("expected single emission for call_abc, got %q", body)
	}
}

func TestChatStreamTranslatorTextOnlyFinishStop(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	tr := newChatStreamTranslator(rec, "test-model", "chatcmpl_test")

	if err := tr.emitTranslatedBlock([]byte(
		"event: response.output_text.delta\n" +
			`data: {"type":"response.output_text.delta","delta":"hello"}`,
	)); err != nil {
		t.Fatal(err)
	}
	if err := tr.emitTranslatedBlock([]byte(
		"event: response.completed\n" +
			`data: {"type":"response.completed","response":{"output":[]}}`,
	)); err != nil {
		t.Fatal(err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"finish_reason":"stop"`) {
		t.Fatalf("expected stop finish, got %q", body)
	}
	if strings.Contains(body, `"tool_calls"`) {
		t.Fatalf("unexpected tool_calls in text-only stream: %q", body)
	}
}

func TestResponsesStreamToolCallFromOutputItemDone(t *testing.T) {
	t.Parallel()

	data := []byte(`{"type":"response.output_item.done","item":{"type":"function_call","call_id":"c1","name":"fn","arguments":"{}"}}`)
	tc, ok := responsesStreamToolCall("response.output_item.done", data)
	if !ok || tc.callID != "c1" || tc.name != "fn" {
		t.Fatalf("unexpected parse: ok=%v tc=%+v", ok, tc)
	}
}

func TestChatStreamTranslatorMidStreamErrorStaysOnChatWire(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	tr := newChatStreamTranslator(rec, "test-model", "chatcmpl_test")

	if err := tr.emitTranslatedBlock([]byte(
		"event: response.output_text.delta\n" +
			`data: {"type":"response.output_text.delta","delta":"hello"}`,
	)); err != nil {
		t.Fatal(err)
	}
	if err := tr.emitTranslatedBlock([]byte(
		"event: error\n" +
			`data: {"type":"error","error":{"message":"upstream boom"}}`,
	)); err != nil {
		t.Fatal(err)
	}

	body := rec.Body.String()
	if strings.Contains(body, `"type":"error"`) && strings.Contains(body, "event:") {
		t.Fatalf("must not mix Responses event framing into chat stream: %q", body)
	}
	if !strings.Contains(body, `"type":"server_error"`) || !strings.Contains(body, "upstream boom") {
		t.Fatalf("expected chat-shaped stream error, got %q", body)
	}
	if !strings.Contains(body, "[DONE]") {
		t.Fatalf("expected DONE after chat stream error, got %q", body)
	}
}
