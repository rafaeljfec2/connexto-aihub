package routing_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestSelectDeterministic_VirtualAuto_Gate0EnforcedBlocks(t *testing.T) {
	prev := routing.AutoGate0Enforced
	t.Cleanup(func() { routing.AutoGate0Enforced = prev })
	routing.AutoGate0Enforced = true

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	snapshot := routing.Snapshot{CapturedAt: now, AutoQuota: routing.BuildAutoQuotaView(nil, now, 2*time.Minute)}
	ctx := routing.RoutingContext{
		RequestID: "req-gate0",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      routing.VirtualModelAuto,
		},
		Understanding: routing.RequestUnderstanding{OptimizationProfile: routing.ProfileBalanced},
	}
	eligibility := routing.EligibilityResult{
		Eligible: []routing.Target{{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"}},
	}

	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)
	if decision.Rejection != routing.RejectRoutingUnavailable {
		t.Fatalf("rejection = %q, want routing_unavailable", decision.Rejection)
	}
	joined := strings.Join(decision.Criteria, ",")
	if !strings.Contains(joined, "gate0_unsatisfied") {
		t.Fatalf("criteria %v missing gate0_unsatisfied", decision.Criteria)
	}
	if decision.Selected != nil {
		t.Fatalf("expected no selection under gate0, got %+v", decision.Selected)
	}

	status, mapped := routing.MapAutoGate0HTTPStatus(decision.Criteria)
	if !mapped || status != 503 {
		t.Fatalf("MapAutoGate0HTTPStatus = (%d, %v), want (503, true)", status, mapped)
	}
}

func TestSelectDeterministic_VirtualAuto_ExcludesBlockedAntigravityHomologation(t *testing.T) {
	prev := routing.AutoGate0Enforced
	t.Cleanup(func() { routing.AutoGate0Enforced = prev })
	routing.AutoGate0Enforced = false

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
	if len(allowlisted) < 7 {
		t.Fatalf("expected at least 7 allowlisted eligible models, got %d: %+v", len(allowlisted), allowlisted)
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
