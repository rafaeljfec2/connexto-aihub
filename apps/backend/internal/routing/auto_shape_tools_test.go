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
		Requirements: routing.RequirementStatus{RequiredCapabilities: []string{"function_tools", "custom_tools"}},
	}
	out, reasons := routing.FilterAutoAntigravityWithoutTools(candidates, understanding)
	if len(out) != 3 {
		t.Fatalf("out=%+v want 3; antigravity stays eligible when tools are required", out)
	}
	found := false
	for _, tgt := range out {
		if tgt.ProviderID == providers.AntigravityID {
			found = true
		}
	}
	if !found {
		t.Fatal("antigravity should remain when tools are required")
	}
	if len(reasons) != 0 {
		t.Fatalf("reasons=%v want none", reasons)
	}
}
