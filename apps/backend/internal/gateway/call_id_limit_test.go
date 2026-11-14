package gateway

import (
	"encoding/json"
	"strings"
	"testing"

	"aihub/internal/providers"
)

func TestLimitResponsesCallIDsShrinksLongIDsTogether(t *testing.T) {
	longID := "call_" + strings.Repeat("a", 101)
	if len(longID) != 106 {
		t.Fatalf("fixture length = %d, want 106", len(longID))
	}
	body := []byte(`{"model":"composer-2.5","input":[` +
		`{"type":"function_call","call_id":"` + longID + `","name":"exec","arguments":"{}"},` +
		`{"type":"function_call_output","call_id":"` + longID + `","output":"ok"}` +
		`]}`)

	out, err := limitResponsesCallIDs(body)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Input []struct {
			CallID string `json:"call_id"`
		} `json:"input"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Input) != 2 {
		t.Fatalf("input len = %d", len(envelope.Input))
	}
	first := envelope.Input[0].CallID
	second := envelope.Input[1].CallID
	if first != second {
		t.Fatalf("paired call ids diverged: %q vs %q", first, second)
	}
	if len(first) > 64 || len(first) == 0 || first == longID {
		t.Fatalf("call_id = %q (len %d)", first, len(first))
	}
}

func TestAdaptResponsesPayloadForProviderLimitsCallIDOnCursor(t *testing.T) {
	longID := "call_" + strings.Repeat("b", 101)
	body := []byte(`{"input":[{"type":"function_call","call_id":"` + longID + `","name":"exec","arguments":"{}"}]}`)
	out, err := adaptResponsesPayloadForProvider(providers.CursorID, body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), longID) {
		t.Fatalf("long call_id survived: %s", out)
	}
}
