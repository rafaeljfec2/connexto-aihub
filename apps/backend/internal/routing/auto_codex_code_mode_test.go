package routing_test

import (
	"testing"
	"time"

	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestSelectDeterministic_AutoKeepsAllowlistForCodexShapedTurn(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	snapshot := routing.Snapshot{
		CapturedAt: now,
		AutoQuota:  routing.BuildAutoQuotaView(nil, now, 2*time.Minute),
	}
	eligible := []routing.Target{
		{ProviderID: providers.CursorID, ModelID: "composer-2.5"},
		{ProviderID: providers.AntigravityID, ModelID: "antigravity:gemini-3.8-flash"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "glm-5.3-flash"},
	}
	ctx := routing.RoutingContext{
		RequestID: "req-auto-codex-shape",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      routing.VirtualModelAuto,
		},
		Understanding: routing.RequestUnderstanding{
			OptimizationProfile: routing.ProfileBalanced,
			Requirements:        routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}},
		},
	}
	eligibility := routing.EligibilityResult{Eligible: eligible}
	seen := map[string]int{}
	for i := 0; i < 12; i++ {
		decision := routing.SelectDeterministic(ctx, snapshot, eligibility)
		if decision.Rejection != "" {
			t.Fatalf("iteration %d: rejection %q criteria=%v", i, decision.Rejection, decision.Criteria)
		}
		if decision.Selected == nil {
			t.Fatalf("iteration %d: missing selection", i)
		}
		if !routing.IsAutoAllowlistTarget(*decision.Selected) {
			t.Fatalf("iteration %d: selected outside allowlist: %+v", i, decision.Selected)
		}
		seen[decision.Selected.ModelID]++
	}
	if len(seen) < 2 {
		t.Fatalf("auto collapsed to one model: %v", seen)
	}
}
