package routing_test

import (
	"testing"

	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestFilterAutoAntigravityWithoutTools(t *testing.T) {
	candidates := []routing.Target{
		{ProviderID: providers.CursorID, ModelID: "composer-2.5"},
		{ProviderID: providers.AntigravityID, ModelID: "antigravity:gemini-3.8-flash"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
	}
	understanding := routing.RequestUnderstanding{
		Requirements: routing.RequirementStatus{RequiredCapabilities: []string{"function_tools"}},
	}
	out, reasons := routing.FilterAutoAntigravityWithoutTools(candidates, understanding)
	if len(out) != 2 {
		t.Fatalf("out=%+v want 2", out)
	}
	for _, tgt := range out {
		if tgt.ProviderID == providers.AntigravityID {
			t.Fatal("antigravity should be excluded when tools are required")
		}
	}
	if len(reasons) == 0 || reasons[0] != "antigravity_tools_unsupported" {
		t.Fatalf("reasons=%v", reasons)
	}
}
