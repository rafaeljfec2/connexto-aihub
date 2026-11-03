package routing_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestSelectDeterministicProviderNotFound(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)

	ctx := routing.RoutingContext{
		RequestID:        "req",
		ExplicitProvider: "unknown-provider",
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Rejection != routing.RejectProviderNotFound {
		t.Fatalf("rejection = %q, want provider_not_found", decision.Rejection)
	}
}

func TestSelectDeterministicExplicitProviderUniqueModel(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)

	ctx := routing.RoutingContext{
		RequestID:        "req",
		ExplicitProvider: "openrouter",
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	assertSelected(t, decision, "openrouter", "model-x", routing.OriginExplicitProvider)
}

func TestSelectDeterministicExplicitProviderMultipleModelsAmbiguous(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)
	snapshot.Models = append(snapshot.Models, modelView("openrouter", "model-y", now, success, providers.StateHealthy))

	ctx := routing.RoutingContext{
		RequestID:        "req",
		ExplicitProvider: "openrouter",
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Rejection != routing.RejectRoutingAmbiguous {
		t.Fatalf("rejection = %q, want routing_ambiguous", decision.Rejection)
	}
}

func TestSelectDeterministicExplicitModelNotFoundWithCompleteCoverage(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)

	ctx := routing.RoutingContext{
		RequestID: "req",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      "does-not-exist",
		},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	if !eligibility.Coverage.CompleteFreshCoverageForEligible {
		t.Fatal("precondition: expected complete fresh coverage")
	}

	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Rejection != routing.RejectModelNotFound {
		t.Fatalf("rejection = %q, want model_not_found", decision.Rejection)
	}
}

func TestSelectDeterministicNoSelectorGlobalIncompleteCoverageUnavailable(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	staleSuccess := now.Add(-48 * time.Hour)
	snapshot := twoProviderSnapshot(now, staleSuccess)
	snapshot.Admin[providers.OpenCodeGoID] = routing.AdminProviderFacts{
		ID: providers.OpenCodeGoID, Known: true, Enabled: false, HasCredential: true,
	}

	eligibility := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(routing.RoutingContext{RequestID: "req"}, snapshot, eligibility)

	if decision.Rejection != routing.RejectRoutingUnavailable {
		t.Fatalf("rejection = %q, want routing_unavailable", decision.Rejection)
	}
}

func TestSelectDeterministicNoSelectorMultipleNonOpenCodeAmbiguous(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)
	snapshot.Admin[providers.OpenCodeGoID] = routing.AdminProviderFacts{
		ID: providers.OpenCodeGoID, Known: true, Enabled: false, HasCredential: true,
	}
	snapshot.Models = append(snapshot.Models, modelView("openrouter", "model-y", now, success, providers.StateHealthy))

	eligibility := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(routing.RoutingContext{RequestID: "req"}, snapshot, eligibility)

	if decision.Rejection != routing.RejectRoutingAmbiguous {
		t.Fatalf("rejection = %q, want routing_ambiguous", decision.Rejection)
	}
}

func TestSelectDeterministicNoEligibleIncompatibleRejection(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	entry := snapshot.Models[0].Entry
	entry.Capabilities["reasoning"] = catalog.CapabilityClaim{State: catalog.CapabilityUnsupported}
	snapshot.Models[0].Entry = entry

	eligibility := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"reasoning"}})
	decision := routing.SelectDeterministic(routing.RoutingContext{RequestID: "req"}, snapshot, eligibility)

	if decision.Rejection != routing.RejectNoEligibleModel {
		t.Fatalf("rejection = %q, want no_eligible_model", decision.Rejection)
	}
}

func TestSelectDeterministicRepeatInputSameDecision(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)
	ctx := routing.RoutingContext{RequestID: "req"}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})

	first := routing.SelectDeterministic(ctx, snapshot, eligibility)
	second := routing.SelectDeterministic(ctx, snapshot, eligibility)

	assertDecisionsEqual(t, first, second)
}

func TestSelectDeterministicShuffledEligibleOrderSameOutcome(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)
	ctx := routing.RoutingContext{RequestID: "req"}

	baseEligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})

	shuffled := baseEligibility
	shuffled.Eligible = []routing.Target{
		{ProviderID: "openrouter", ModelID: "model-x"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "model-a"},
	}

	ordered := baseEligibility
	ordered.Eligible = []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "model-a"},
		{ProviderID: "openrouter", ModelID: "model-x"},
	}

	first := routing.SelectDeterministic(ctx, snapshot, shuffled)
	second := routing.SelectDeterministic(ctx, snapshot, ordered)

	assertDecisionsEqual(t, first, second)
	assertSelected(t, first, providers.OpenCodeGoID, "model-a", routing.OriginCompatibilityDefault)
}

func TestSelectDeterministicShuffledEligibleAmbiguousSameRejection(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := routing.Snapshot{
		CapturedAt: now,
		StaleAfter: 24 * time.Hour,
		Admin: map[string]routing.AdminProviderFacts{
			providers.OpenCodeGoID: {ID: providers.OpenCodeGoID, Known: true, Enabled: true, HasCredential: true},
		},
		Runtime: map[string]routing.RuntimeProviderFacts{
			providers.OpenCodeGoID: {Bound: true, ResponsesSupported: true, RuntimeEnabled: true},
		},
		Sync: map[string]catalog.ProviderSyncState{
			providers.OpenCodeGoID: {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
		},
		Models: []catalog.View{
			modelView(providers.OpenCodeGoID, "model-a", now, success, providers.StateHealthy),
			modelView(providers.OpenCodeGoID, "model-b", now, success, providers.StateHealthy),
		},
	}

	ctx := routing.RoutingContext{RequestID: "req"}
	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})

	reversed := eligibility
	reversed.Eligible = []routing.Target{
		eligibility.Eligible[1],
		eligibility.Eligible[0],
	}

	first := routing.SelectDeterministic(ctx, snapshot, eligibility)
	second := routing.SelectDeterministic(ctx, snapshot, reversed)

	if first.Rejection != routing.RejectRoutingAmbiguous || second.Rejection != routing.RejectRoutingAmbiguous {
		t.Fatalf("rejections = %q and %q, want routing_ambiguous", first.Rejection, second.Rejection)
	}

	if first.Selected != nil || second.Selected != nil {
		t.Fatal("expected no selection")
	}
}

func TestSelectDeterministicDoesNotMutateEligibilityInput(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)
	ctx := routing.RoutingContext{RequestID: "req"}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	cloneEligible := append([]routing.Target(nil), eligibility.Eligible...)

	_ = routing.SelectDeterministic(ctx, snapshot, eligibility)

	if len(eligibility.Eligible) != len(cloneEligible) {
		t.Fatalf("eligible len changed: %d -> %d", len(cloneEligible), len(eligibility.Eligible))
	}

	for i := range cloneEligible {
		if eligibility.Eligible[i] != cloneEligible[i] {
			t.Fatalf("eligible[%d] mutated", i)
		}
	}
}

func assertDecisionsEqual(t *testing.T, left, right routing.RoutingDecision) {
	t.Helper()

	if left.Rejection != right.Rejection {
		t.Fatalf("rejections differ: %q vs %q", left.Rejection, right.Rejection)
	}

	if left.Origin != right.Origin {
		t.Fatalf("origins differ: %q vs %q", left.Origin, right.Origin)
	}

	if (left.Selected == nil) != (right.Selected == nil) {
		t.Fatalf("selection presence differs: %+v vs %+v", left.Selected, right.Selected)
	}

	if left.Selected != nil && *left.Selected != *right.Selected {
		t.Fatalf("selected targets differ: %+v vs %+v", left.Selected, right.Selected)
	}
}
