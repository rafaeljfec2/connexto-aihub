package capture_test

import (
	"testing"

	"aihub/internal/metering/capture"
)

func TestStreamTapKeepsLastResponseCompletedUsage(t *testing.T) {
	tap := capture.NewStreamTap()
	events := []string{
		"event: response.output_text.delta\ndata: {\"delta\":\"hi\"}\n\n",
		"event: response.completed\ndata: {\"usage\":{\"input_tokens\":1}}\n\n",
		"event: response.completed\ndata: {\"usage\":{\"input_tokens\":9,\"output_tokens\":4,\"total_tokens\":13}}\n\n",
	}
	for _, event := range events {
		tap.Observe([]byte(event))
	}
	result := tap.Finalize("text/event-stream")
	if len(result.FinalEventData) == 0 {
		t.Fatal("expected final event data")
	}
	if string(result.FinalEventData) != `{"usage":{"input_tokens":9,"output_tokens":4,"total_tokens":13}}` {
		t.Fatalf("final event = %s", result.FinalEventData)
	}
}

func TestStreamTapCapturesOpenRouterResponseDone(t *testing.T) {
	tap := capture.NewStreamTap()
	payload := "data: {\"type\":\"response.done\",\"response\":{\"usage\":{\"input_tokens\":2,\"output_tokens\":3,\"total_tokens\":5}}}\n\n"
	tap.Observe([]byte(payload))
	result := tap.Finalize("text/event-stream")
	if string(result.FinalEventData) != `{"type":"response.done","response":{"usage":{"input_tokens":2,"output_tokens":3,"total_tokens":5}}}` {
		t.Fatalf("final event = %s", result.FinalEventData)
	}
}

func TestSSETerminalObserverIgnoresProseMentioningCompleted(t *testing.T) {
	var obs capture.SSETerminalObserver
	obs.ObserveChunk([]byte(
		"event: response.output_text.delta\n" +
			`data: {"type":"response.output_text.delta","delta":"see response.completed in the docs"}` + "\n\n",
	))
	if obs.SawTerminal {
		t.Fatal("prose in delta must not count as terminal event")
	}
	obs.ObserveChunk([]byte("event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n"))
	if !obs.SawTerminal {
		t.Fatal("expected real terminal event to be detected")
	}
}

func TestStreamTapIgnoresIntermediateEvents(t *testing.T) {
	tap := capture.NewStreamTap()
	tap.Observe([]byte("data: {\"type\":\"response.created\"}\n\n"))
	result := tap.Finalize("text/event-stream")
	if len(result.FinalEventData) != 0 {
		t.Fatalf("unexpected final event = %s", result.FinalEventData)
	}
}

func TestJSONBodyUsageTail(t *testing.T) {
	tap := capture.NewStreamTap()
	tap.Observe([]byte(`{"id":"resp","usage":{"input_tokens":4,"output_tokens":6,"total_tokens":10}}`))
	result := tap.Finalize("application/json")
	if !result.Streamed {
		if len(result.ResponseBodyTail) == 0 {
			t.Fatal("expected body tail")
		}
	}
}
