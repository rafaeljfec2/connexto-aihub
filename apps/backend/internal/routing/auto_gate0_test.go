package routing_test

import (
	"slices"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestAutoGate0Satisfied_AllowlistApproved(t *testing.T) {
	if !routing.AutoGate0Satisfied() {
		t.Fatal("expected Gate 0 satisfied for all allowlist models after 2026-11-12 battery")
	}
}

func TestSelectDeterministic_VirtualAuto_ExcludesBlockedAntigravityHomologation(t *testing.T) {
	blocked := []string{
		"antigravity:claude-opus-4-6-thinking",
		"antigravity:claude-sonnet-4-6",
		"antigravity:gpt-oss-120b-medium",
	}

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	providersFacts := map[string]routing.AdminProviderFacts{
		providers.OpenCodeGoID:   {ID: providers.OpenCodeGoID, Known: true, Enabled: true, HasCredential: true},
		providers.OpenRouterID:   {ID: providers.OpenRouterID, Known: true, Enabled: true, HasCredential: true},
		providers.CursorID:       {ID: providers.CursorID, Known: true, Enabled: true, HasCredential: true},
		providers.AntigravityID:  {ID: providers.AntigravityID, Known: true, Enabled: true, HasCredential: true},
	}
	runtimeFacts := map[string]routing.RuntimeProviderFacts{
		providers.OpenCodeGoID:  {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
		providers.OpenRouterID:  {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
		providers.CursorID:      {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
		providers.AntigravityID: {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
	}
	syncState := map[string]catalog.ProviderSyncState{
		providers.OpenCodeGoID:  {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
		providers.OpenRouterID:  {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
		providers.CursorID:      {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
		providers.AntigravityID: {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
	}

	models := make([]catalog.View, 0, len(routing.AutoMVPAllowlist))
	for _, target := range routing.AutoMVPAllowlist {
		models = append(models, autoGateModelView(target.ProviderID, target.ModelID, now, success))
	}

	snapshot := routing.Snapshot{
		CapturedAt: now,
		StaleAfter: 24 * time.Hour,
		Admin:      providersFacts,
		Runtime:    runtimeFacts,
		Sync:       syncState,
		Models:     models,
		AutoQuota:  routing.BuildAutoQuotaView(nil, now, 2*time.Minute),
	}

	ctx := routing.RoutingContext{
		RequestID: "req-auto-reasoning",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      routing.VirtualModelAuto,
		},
		Understanding: routing.RequestUnderstanding{
			OptimizationProfile: routing.ProfileBalanced,
			Requirements: routing.RequirementStatus{
				RequiredCapabilities: []string{"reasoning"},
			},
		},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, ctx.Understanding.Requirements)
	allowlisted := routing.FilterEligibleToAutoAllowlist(eligibility.Eligible)
	if len(allowlisted) < len(routing.AutoMVPAllowlist) {
		t.Fatalf("expected %d allowlisted eligible models, got %d: %+v", len(routing.AutoMVPAllowlist), len(allowlisted), allowlisted)
	}

	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)
	if decision.Rejection != "" {
		t.Fatalf("expected selection, got rejection %q criteria=%v", decision.Rejection, decision.Criteria)
	}
	if decision.Selected == nil {
		t.Fatal("expected selected target")
	}
	if slices.Contains(blocked, decision.Selected.ModelID) {
		t.Fatalf("selected blocked antigravity model %q", decision.Selected.ModelID)
	}
	if !routing.IsAutoAllowlistTarget(*decision.Selected) {
		t.Fatalf("selected outside allowlist: %+v", decision.Selected)
	}
}

func TestSelectDeterministic_VirtualAuto_BandBBeforePoolAvoidsDeadAntigravityPool(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	snapshot := routing.Snapshot{
		CapturedAt: now,
		AutoQuota:  routing.BuildAutoQuotaView(nil, now, 2*time.Minute),
	}
	blockedOnly := []routing.Target{
		{ProviderID: providers.AntigravityID, ModelID: "antigravity:claude-opus-4-6-thinking"},
		{ProviderID: providers.AntigravityID, ModelID: "antigravity:claude-sonnet-4-6"},
		{ProviderID: providers.AntigravityID, ModelID: "antigravity:gpt-oss-120b-medium"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
	}
	ctx := routing.RoutingContext{
		RequestID: "req-bandb-pool",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      routing.VirtualModelAuto,
		},
		Understanding: routing.RequestUnderstanding{
			OptimizationProfile: routing.ProfileBalanced,
			Requirements:        routing.RequirementStatus{RequiredCapabilities: []string{"reasoning"}},
		},
	}
	eligibility := routing.EligibilityResult{Eligible: blockedOnly}
	for i := 0; i < 12; i++ {
		decision := routing.SelectDeterministic(ctx, snapshot, eligibility)
		if decision.Rejection != "" {
			t.Fatalf("iteration %d: rejection %q criteria=%v", i, decision.Rejection, decision.Criteria)
		}
		if decision.Selected == nil || decision.Selected.ModelID != "gpt-6-luna" {
			t.Fatalf("iteration %d: selected %+v, want gpt-6-luna", i, decision.Selected)
		}
	}
}

func autoGateModelView(providerID, modelID string, now, success time.Time) catalog.View {
	entry := catalog.ModelEntry{
		ProviderID: providerID,
		ModelID:    modelID,
		Capabilities: map[string]catalog.CapabilityClaim{
			"text_generation": {State: catalog.CapabilitySupported},
			"reasoning":       {State: catalog.CapabilitySupported},
		},
	}
	sync := catalog.ProviderSyncState{LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess}
	return catalog.DeriveView(entry, sync, providers.StateHealthy, now, 24*time.Hour)
}
