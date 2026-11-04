package capture_test

import (
	"testing"

	"aihub/internal/metering/capture"
)

func TestExtractFinishReasonFromResponsesCompletedStatus(t *testing.T) {
	data := []byte(`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":1}}}`)
	reason := capture.ExtractFinishReason(capture.Result{FinalEventData: data})
	if reason == nil || *reason != "completed" {
		t.Fatalf("finish reason = %v, want completed", reason)
	}
}

func TestExtractFinishReasonPrefersIncompleteDetails(t *testing.T) {
	data := []byte(`{"type":"response.completed","response":{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`)
	reason := capture.ExtractFinishReason(capture.Result{FinalEventData: data})
	if reason == nil || *reason != "max_output_tokens" {
		t.Fatalf("finish reason = %v, want max_output_tokens", reason)
	}
}

func TestExtractFinishReasonFromChatStyleChoices(t *testing.T) {
	data := []byte(`{"choices":[{"finish_reason":"stop","native_finish_reason":"STOP"}]}`)
	reason := capture.ExtractFinishReason(capture.Result{FinalEventData: data})
	if reason == nil || *reason != "stop" {
		t.Fatalf("finish reason = %v, want stop", reason)
	}
}

func TestExtractFinishReasonAbsentWhenPayloadMissing(t *testing.T) {
	if reason := capture.ExtractFinishReason(capture.Result{}); reason != nil {
		t.Fatalf("finish reason = %v, want nil", reason)
	}
}
