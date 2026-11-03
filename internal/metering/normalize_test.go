package metering_test

import (
	"testing"

	"aihub/internal/metering"
	"aihub/internal/metering/capture"
	"aihub/internal/providers"
)

func TestNormalizeOpenCodeUsageReported(t *testing.T) {
	event := []byte(`{"usage":{"input_tokens":36,"output_tokens":19,"total_tokens":55,"cost":0,"opencode":{"tokens":{"cache_read":0}}}}`)
	record := metering.NormalizeExecutionUsage(providers.OpenCodeGoID, capture.Result{FinalEventData: event})
	if record.Status != metering.StatusReported {
		t.Fatalf("status = %s", record.Status)
	}
	if record.InputTokens == nil || *record.InputTokens != 36 {
		t.Fatalf("input = %v", record.InputTokens)
	}
}

func TestNormalizeOpenCodeUsageUnavailable(t *testing.T) {
	record := metering.NormalizeExecutionUsage(providers.OpenCodeGoID, capture.Result{})
	if record.Status != metering.StatusUnavailable {
		t.Fatalf("status = %s", record.Status)
	}
}

func TestNormalizeOpenCodeUsagePartial(t *testing.T) {
	event := []byte(`{"usage":{"input_tokens":1}}`)
	record := metering.NormalizeExecutionUsage(providers.OpenCodeGoID, capture.Result{FinalEventData: event})
	if record.Status != metering.StatusPartial {
		t.Fatalf("status = %s", record.Status)
	}
}

func TestNormalizeOpenCodeUsageInvalidTotals(t *testing.T) {
	event := []byte(`{"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":9}}`)
	record := metering.NormalizeExecutionUsage(providers.OpenCodeGoID, capture.Result{FinalEventData: event})
	if record.Status != metering.StatusInvalid {
		t.Fatalf("status = %s", record.Status)
	}
}

func TestNormalizeOpenRouterUsageFromResponseDone(t *testing.T) {
	event := []byte(`{"type":"response.done","response":{"usage":{"input_tokens":12,"output_tokens":45,"total_tokens":57}}}`)
	record := metering.NormalizeExecutionUsage(providers.OpenRouterID, capture.Result{FinalEventData: event})
	if record.Status != metering.StatusReported {
		t.Fatalf("status = %s", record.Status)
	}
	if record.Source != metering.SourceOpenRouter {
		t.Fatalf("source = %s", record.Source)
	}
}

func TestNormalizeExplicitZeroTokensAreReported(t *testing.T) {
	event := []byte(`{"usage":{"input_tokens":0,"output_tokens":0,"total_tokens":0}}`)
	record := metering.NormalizeExecutionUsage(providers.OpenRouterID, capture.Result{FinalEventData: event})
	if record.Status != metering.StatusReported {
		t.Fatalf("status = %s", record.Status)
	}
	if record.InputTokens == nil || *record.InputTokens != 0 {
		t.Fatalf("input = %v", record.InputTokens)
	}
}
