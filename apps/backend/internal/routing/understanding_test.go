package routing_test

import (
	"testing"

	"aihub/internal/routing"
)

func TestIsVirtualModel(t *testing.T) {
	tests := []struct {
		modelID string
		want    bool
	}{
		{modelID: "aihub/auto", want: true},
		{modelID: "openrouter/auto", want: true},
		{modelID: "gpt-6-luna", want: false},
		{modelID: "anthropic/claude-haiku-4.5", want: false},
		{modelID: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := routing.IsVirtualModel(tt.modelID)
			if got != tt.want {
				t.Errorf("IsVirtualModel(%q) = %v, want %v", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestUnderstandRequest_VirtualAuto(t *testing.T) {
	body := []byte(`{
		"model": "aihub/auto",
		"input": "Explain the difference between a queue and a stream.",
		"stream": true,
		"max_output_tokens": 1024
	}`)

	structural, err := routing.ValidateStructure(body)
	if err != nil {
		t.Fatalf("ValidateStructure failed: %v", err)
	}

	understanding, parseErr := routing.UnderstandRequest(body, structural)
	if parseErr != nil {
		t.Fatalf("UnderstandRequest failed: %v", parseErr)
	}

	if !understanding.IsVirtual {
		t.Errorf("expected IsVirtual = true, got false")
	}

	if understanding.VirtualTarget != routing.VirtualModelAuto {
		t.Errorf("expected VirtualTarget = %q, got %q", routing.VirtualModelAuto, understanding.VirtualTarget)
	}

	if !understanding.IsStreaming {
		t.Errorf("expected IsStreaming = true, got false")
	}

	if understanding.OptimizationProfile != routing.ProfileBalanced {
		t.Errorf("expected OptimizationProfile = %q, got %q", routing.ProfileBalanced, understanding.OptimizationProfile)
	}

	if understanding.EstimatedInputTokens <= 0 {
		t.Errorf("expected EstimatedInputTokens > 0, got %d", understanding.EstimatedInputTokens)
	}

	foundTextGen := false
	for _, cap := range understanding.Requirements.RequiredCapabilities {
		if cap == "text_generation" {
			foundTextGen = true
			break
		}
	}
	if !foundTextGen {
		t.Errorf("expected required capability 'text_generation' in %+v", understanding.Requirements.RequiredCapabilities)
	}
}

func TestUnderstandRequest_ExplicitModel(t *testing.T) {
	body := []byte(`{
		"model": "anthropic/claude-haiku-4.5",
		"input": "Hello world"
	}`)

	structural, err := routing.ValidateStructure(body)
	if err != nil {
		t.Fatalf("ValidateStructure failed: %v", err)
	}

	understanding, parseErr := routing.UnderstandRequest(body, structural)
	if parseErr != nil {
		t.Fatalf("UnderstandRequest failed: %v", parseErr)
	}

	if understanding.IsVirtual {
		t.Errorf("expected IsVirtual = false, got true")
	}

	if understanding.IsStreaming {
		t.Errorf("expected IsStreaming = false, got true")
	}
}

func TestUnderstandRequest_EffortOnlyDoesNotRequireReasoning(t *testing.T) {
	body := []byte(`{
		"model": "antigravity:gpt-oss-120b-medium",
		"input": "oi",
		"reasoning": {"effort": "medium", "context": "all_turns"}
	}`)
	structural, err := routing.ValidateStructure(body)
	if err != nil {
		t.Fatalf("ValidateStructure failed: %v", err)
	}
	understanding, parseErr := routing.UnderstandRequest(body, structural)
	if parseErr != nil {
		t.Fatalf("UnderstandRequest failed: %v", parseErr)
	}
	for _, cap := range understanding.Requirements.RequiredCapabilities {
		if cap == "reasoning" {
			t.Fatalf("effort-only reasoning required capability: %+v", understanding.Requirements.RequiredCapabilities)
		}
	}
}

func TestUnderstandRequest_ReasoningBudgetRequiresCapability(t *testing.T) {
	body := []byte(`{"model":"x","input":"oi","reasoning":{"effort":"high","max_tokens":2000}}`)
	structural, err := routing.ValidateStructure(body)
	if err != nil {
		t.Fatalf("ValidateStructure failed: %v", err)
	}
	understanding, parseErr := routing.UnderstandRequest(body, structural)
	if parseErr != nil {
		t.Fatalf("UnderstandRequest failed: %v", parseErr)
	}
	found := false
	for _, cap := range understanding.Requirements.RequiredCapabilities {
		if cap == "reasoning" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected reasoning capability, got %+v", understanding.Requirements.RequiredCapabilities)
	}
}

func TestUnderstandRequest_ToolsAndSchemaCapabilities(t *testing.T) {
	body := []byte(`{
		"model": "aihub/auto",
		"input": "Analyze this data",
		"tools": [
			{"type": "function", "name": "search_db"}
		],
		"text": {
			"format": {"type": "json_schema"}
		}
	}`)

	structural, err := routing.ValidateStructure(body)
	if err != nil {
		t.Fatalf("ValidateStructure failed: %v", err)
	}

	understanding, parseErr := routing.UnderstandRequest(body, structural)
	if parseErr != nil {
		t.Fatalf("UnderstandRequest failed: %v", parseErr)
	}

	caps := map[string]bool{}
	for _, c := range understanding.Requirements.RequiredCapabilities {
		caps[c] = true
	}

	if !caps["function_tools"] {
		t.Errorf("expected capability 'function_tools', got %+v", understanding.Requirements.RequiredCapabilities)
	}

	if !caps["structured_output"] {
		t.Errorf("expected capability 'structured_output', got %+v", understanding.Requirements.RequiredCapabilities)
	}
}

func TestUnderstandRequest_ReasoningEffortAndInputItemCount(t *testing.T) {
	body := []byte(`{
		"model": "aihub/auto",
		"input": [
			{"type": "input_text", "text": "a"},
			{"type": "input_text", "text": "b"}
		],
		"reasoning": {"effort": "high"}
	}`)
	structural, err := routing.ValidateStructure(body)
	if err != nil {
		t.Fatalf("ValidateStructure failed: %v", err)
	}
	understanding, parseErr := routing.UnderstandRequest(body, structural)
	if parseErr != nil {
		t.Fatalf("UnderstandRequest failed: %v", parseErr)
	}
	if understanding.ReasoningEffort != routing.ReasoningEffortHigh {
		t.Fatalf("ReasoningEffort = %q, want high", understanding.ReasoningEffort)
	}
	if understanding.InputItemCount != 2 {
		t.Fatalf("InputItemCount = %d, want 2", understanding.InputItemCount)
	}
}

func TestUnderstandRequest_MediumEffortCaptured(t *testing.T) {
	body := []byte(`{"model":"aihub/auto","input":"x","reasoning":{"effort":"medium"}}`)
	structural, err := routing.ValidateStructure(body)
	if err != nil {
		t.Fatalf("ValidateStructure failed: %v", err)
	}
	understanding, parseErr := routing.UnderstandRequest(body, structural)
	if parseErr != nil {
		t.Fatalf("UnderstandRequest failed: %v", parseErr)
	}
	if understanding.ReasoningEffort != routing.ReasoningEffortMedium {
		t.Fatalf("ReasoningEffort = %q, want medium", understanding.ReasoningEffort)
	}
}
