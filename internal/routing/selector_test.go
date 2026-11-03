package routing_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestSelectDeterministicCompatibilityDefaultSingleOpenCodeModel(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)

	eligibility := routing.EvaluateEligibility(routing.RoutingContext{
		RequestID: "req-1",
	}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})

	decision := routing.SelectDeterministic(routing.RoutingContext{RequestID: "req-1"}, snapshot, eligibility)
	assertSelected(t, decision, providers.OpenCodeGoID, "model-a", routing.OriginCompatibilityDefault)
}

func TestSelectDeterministicCompatibilityDefaultTwoOpenCodeModelsAmbiguous(t *testing.T) {
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

	eligibility := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(routing.RoutingContext{RequestID: "req"}, snapshot, eligibility)

	if decision.Rejection != routing.RejectRoutingAmbiguous {
		t.Fatalf("rejection = %q, want routing_ambiguous", decision.Rejection)
	}
}

func TestSelectDeterministicAutomaticUniqueCandidateWhenOpenCodeEmpty(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)
	snapshot.Admin[providers.OpenCodeGoID] = routing.AdminProviderFacts{
		ID: providers.OpenCodeGoID, Known: true, Enabled: false, HasCredential: true,
	}

	eligibility := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(routing.RoutingContext{RequestID: "req"}, snapshot, eligibility)

	assertSelected(t, decision, "openrouter", "model-x", routing.OriginAutomaticUniqueCandidate)
}

func TestSelectDeterministicExplicitModelUniqueMatch(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)

	ctx := routing.RoutingContext{
		RequestID: "req",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      "model-x",
		},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	assertSelected(t, decision, "openrouter", "model-x", routing.OriginExplicitModel)
}

func TestSelectDeterministicExplicitModelAmbiguousAcrossProviders(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)
	snapshot.Models = append(snapshot.Models, modelView(providers.OpenCodeGoID, "shared-model", now, success, providers.StateHealthy))
	snapshot.Models = append(snapshot.Models, modelView("openrouter", "shared-model", now, success, providers.StateHealthy))

	ctx := routing.RoutingContext{
		RequestID: "req",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      "shared-model",
		},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Rejection != routing.RejectRoutingAmbiguous {
		t.Fatalf("rejection = %q, want routing_ambiguous", decision.Rejection)
	}
}

func TestSelectDeterministicExplicitModelUnavailableWhenCoverageIncomplete(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	staleSuccess := now.Add(-48 * time.Hour)
	snapshot := twoProviderSnapshot(now, staleSuccess)

	ctx := routing.RoutingContext{
		RequestID: "req",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      "model-x",
		},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Rejection != routing.RejectRoutingUnavailable {
		t.Fatalf("rejection = %q, want routing_unavailable", decision.Rejection)
	}
}

func TestSelectDeterministicProviderModelConflict(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)

	ctx := routing.RoutingContext{
		RequestID:        "req",
		ExplicitProvider: providers.OpenCodeGoID,
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      "model-x",
		},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Rejection != routing.RejectProviderModelConflict {
		t.Fatalf("rejection = %q, want provider_model_conflict", decision.Rejection)
	}
}

func TestSelectDeterministicExplicitProviderAndModel(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)
	snapshot := twoProviderSnapshot(now, success)

	ctx := routing.RoutingContext{
		RequestID:        "req",
		ExplicitProvider: "openrouter",
		Structural: routing.StructuralRequest{
			ModelPresent: true,
			ModelID:      "model-x",
		},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	assertSelected(t, decision, "openrouter", "model-x", routing.OriginExplicitProviderAndModel)
}

func TestSelectDeterministicRequirementUnmapped(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)

	eligibility := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{Unmapped: true})
	decision := routing.SelectDeterministic(routing.RoutingContext{RequestID: "req"}, snapshot, eligibility)

	if decision.Rejection != routing.RejectRequirementUnmapped {
		t.Fatalf("rejection = %q, want requirement_unmapped", decision.Rejection)
	}
}

func TestSelectDeterministicDoesNotPickFirstSortedModel(t *testing.T) {
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
			modelView(providers.OpenCodeGoID, "aaa", now, success, providers.StateHealthy),
			modelView(providers.OpenCodeGoID, "zzz", now, success, providers.StateHealthy),
		},
	}

	eligibility := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(routing.RoutingContext{RequestID: "req"}, snapshot, eligibility)

	if decision.Selected != nil {
		t.Fatalf("selected = %+v, want rejection", decision.Selected)
	}

	if decision.Rejection != routing.RejectRoutingAmbiguous {
		t.Fatalf("rejection = %q, want routing_ambiguous", decision.Rejection)
	}
}

func twoProviderSnapshot(now, success time.Time) routing.Snapshot {
	return routing.Snapshot{
		CapturedAt: now,
		StaleAfter: 24 * time.Hour,
		Admin: map[string]routing.AdminProviderFacts{
			providers.OpenCodeGoID: {ID: providers.OpenCodeGoID, Known: true, Enabled: true, HasCredential: true},
			"openrouter":           {ID: "openrouter", Known: true, Enabled: true, HasCredential: true},
		},
		Runtime: map[string]routing.RuntimeProviderFacts{
			providers.OpenCodeGoID: {Bound: true, ResponsesSupported: true, RuntimeEnabled: true},
			"openrouter":             {Bound: true, ResponsesSupported: true, RuntimeEnabled: true},
		},
		Sync: map[string]catalog.ProviderSyncState{
			providers.OpenCodeGoID: {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
			"openrouter":             {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
		},
		Models: []catalog.View{
			modelView(providers.OpenCodeGoID, "model-a", now, success, providers.StateHealthy),
			modelView("openrouter", "model-x", now, success, providers.StateHealthy),
		},
	}
}

func assertSelected(t *testing.T, decision routing.RoutingDecision, providerID, modelID string, origin routing.SelectionOrigin) {
	t.Helper()

	if decision.Selected == nil {
		t.Fatalf("expected selection, got rejection %q", decision.Rejection)
	}

	if decision.Selected.ProviderID != providerID || decision.Selected.ModelID != modelID {
		t.Fatalf("selected = %+v, want %s/%s", decision.Selected, providerID, modelID)
	}

	if decision.Origin != origin {
		t.Fatalf("origin = %q, want %q", decision.Origin, origin)
	}
}
