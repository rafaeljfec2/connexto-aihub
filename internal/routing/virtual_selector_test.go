package routing_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestSelectDeterministic_VirtualAuto_SelectsBestCandidate(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)

	snapshot := routing.Snapshot{
		CapturedAt: now,
		StaleAfter: 24 * time.Hour,
		Admin: map[string]routing.AdminProviderFacts{
			providers.OpenCodeGoID: {ID: providers.OpenCodeGoID, Known: true, Enabled: true, HasCredential: true},
			"openrouter":           {ID: "openrouter", Known: true, Enabled: true, HasCredential: true},
		},
		Runtime: map[string]routing.RuntimeProviderFacts{
			providers.OpenCodeGoID: {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
			"openrouter":           {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
		},
		Sync: map[string]catalog.ProviderSyncState{
			providers.OpenCodeGoID: {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
			"openrouter":           {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
		},
		Models: []catalog.View{
			modelView(providers.OpenCodeGoID, "gpt-6-luna", now, success, providers.StateHealthy),
			modelView("openrouter", "anthropic/claude-haiku-4.5", now, success, providers.StateHealthy),
		},
		AutoQuota: routing.BuildAutoQuotaView(nil, now, 2*time.Minute),
	}

	ctx := routing.RoutingContext{
		RequestID: "req-auto-1",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      routing.VirtualModelAuto,
		},
		Understanding: routing.RequestUnderstanding{OptimizationProfile: routing.ProfileBalanced},
	}

	reqStatus := routing.RequirementStatus{
		RequiredCapabilities: []string{"text_generation"},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, reqStatus)
	if len(eligibility.Eligible) != 2 {
		t.Fatalf("expected 2 eligible models, got %d", len(eligibility.Eligible))
	}

	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Rejection != "" {
		t.Fatalf("expected decision not to be rejected, got %v", decision.Rejection)
	}

	if decision.Selected == nil {
		t.Fatalf("expected selected target, got nil")
	}

	if decision.Origin != routing.OriginVirtualAuto {
		t.Errorf("expected Origin = %q, got %q", routing.OriginVirtualAuto, decision.Origin)
	}

	if !routing.IsAutoAllowlistTarget(*decision.Selected) {
		t.Errorf("selected model %q is outside the auto allowlist", decision.Selected.ModelID)
	}
	if decision.RoutingReasonCode != "awrr_pool_select" {
		t.Errorf("expected awrr_pool_select, got %q", decision.RoutingReasonCode)
	}
}

func TestSelectDeterministic_VirtualAuto_NoEligibleModels(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	snapshot := routing.Snapshot{
		CapturedAt: now,
		StaleAfter: 24 * time.Hour,
		Admin:      map[string]routing.AdminProviderFacts{},
		Runtime:    map[string]routing.RuntimeProviderFacts{},
		Models:     []catalog.View{},
		Sync:       map[string]catalog.ProviderSyncState{},
	}

	ctx := routing.RoutingContext{
		RequestID: "req-auto-2",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      routing.VirtualModelAuto,
		},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Selected != nil {
		t.Fatalf("expected selected = nil, got %+v", decision.Selected)
	}

	if decision.Rejection == "" {
		t.Fatalf("expected rejection code, got empty")
	}
}
