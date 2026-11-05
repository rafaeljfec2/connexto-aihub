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
	if record.CachedInputTokens == nil || *record.CachedInputTokens != 0 {
		t.Fatalf("cached input = %v, want explicit zero", record.CachedInputTokens)
	}
	if !record.ProviderReportedCost.Valid || record.Currency != "USD" {
		t.Fatalf("reported cost = %+v currency = %q, want zero USD", record.ProviderReportedCost, record.Currency)
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

func TestNormalizeOpenCodeNullAndAbsentTokensRemainUnknown(t *testing.T) {
	event := []byte(`{"usage":{"input_tokens":null,"output_tokens":0,"total_tokens":null}}`)
	record := metering.NormalizeExecutionUsage(providers.OpenCodeGoID, capture.Result{FinalEventData: event})
	if record.Status != metering.StatusPartial {
		t.Fatalf("status = %s, want partial", record.Status)
	}
	if record.InputTokens != nil || record.TotalTokens != nil {
		t.Fatalf("unknown tokens = input %v, total %v", record.InputTokens, record.TotalTokens)
	}
	if record.CachedInputTokens != nil {
		t.Fatalf("absent cached input = %v, want unknown", record.CachedInputTokens)
	}
	if record.OutputTokens == nil || *record.OutputTokens != 0 {
		t.Fatalf("output = %v, want explicit zero", record.OutputTokens)
	}
}

func TestNormalizeOpenCodeListCostFromOpenCodeMetadata(t *testing.T) {
	event := []byte(`{"usage":{"input_tokens":36,"output_tokens":19,"total_tokens":55,"cost":0,"opencode":{"list_cost":0.0000168,"tokens":{"cache_read":0}}}}`)
	record := metering.NormalizeExecutionUsage(providers.OpenCodeGoID, capture.Result{FinalEventData: event})
	if !record.ProviderReportedCost.Valid || !record.ProviderListCost.Valid {
		t.Fatalf("reported cost = %+v list cost = %+v", record.ProviderReportedCost, record.ProviderListCost)
	}
}

func TestNormalizeOpenRouterCachedTokensAndMissingCurrency(t *testing.T) {
	event := []byte(`{"type":"response.done","response":{"usage":{"input_tokens":12,"output_tokens":45,"total_tokens":57,"cached_input_tokens":3,"cost":0}}}`)
	record := metering.NormalizeExecutionUsage(providers.OpenRouterID, capture.Result{FinalEventData: event})
	if record.Status != metering.StatusReported {
		t.Fatalf("status = %s", record.Status)
	}
	if record.CachedInputTokens == nil || *record.CachedInputTokens != 3 {
		t.Fatalf("cached input = %v, want 3", record.CachedInputTokens)
	}
	if !record.ProviderReportedCost.Valid || record.Currency != "USD" {
		t.Fatalf("reported cost = %+v currency = %q, want zero USD", record.ProviderReportedCost, record.Currency)
	}
}

func TestNormalizeConnextoProxyUsageReported(t *testing.T) {
	event := []byte(`{"type":"response.completed","response":{"usage":{"input_tokens":42,"output_tokens":88,"total_tokens":130}}}`)
	record := metering.NormalizeExecutionUsage(providers.ConnextoProxyID, capture.Result{FinalEventData: event})
	if record.Status != metering.StatusReported {
		t.Fatalf("status = %s", record.Status)
	}
	if record.Source != metering.SourceConnextoProxy {
		t.Fatalf("source = %s, want %s", record.Source, metering.SourceConnextoProxy)
	}
	if record.InputTokens == nil || *record.InputTokens != 42 {
		t.Fatalf("input = %v, want 42", record.InputTokens)
	}
	if record.OutputTokens == nil || *record.OutputTokens != 88 {
		t.Fatalf("output = %v, want 88", record.OutputTokens)
	}
	if record.TotalTokens == nil || *record.TotalTokens != 130 {
		t.Fatalf("total = %v, want 130", record.TotalTokens)
	}
}
