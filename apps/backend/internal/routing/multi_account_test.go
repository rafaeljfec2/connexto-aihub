package routing_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestSelectDeterministicTieBreaksMultipleAccountsOfSameTypeWithAWRR(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	highPressure := 0.85
	lowPressure := 0.15
	lowWeight := 0.15
	highWeight := 0.85

	snapshot := routing.Snapshot{
		CapturedAt: now,
		StaleAfter: 10 * time.Minute,
		Admin: map[string]routing.AdminProviderFacts{
			"cursor-1": {ID: "cursor-1", Type: providers.TypeCursor, Known: true, Enabled: true, HasCredential: true},
			"cursor-2": {ID: "cursor-2", Type: providers.TypeCursor, Known: true, Enabled: true, HasCredential: true},
		},
		Runtime: map[string]routing.RuntimeProviderFacts{
			"cursor-1": {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
			"cursor-2": {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
		},
		Sync: map[string]catalog.ProviderSyncState{
			"cursor-1": {Outcome: catalog.SyncOutcomeSuccess, LastSuccessAt: &now},
			"cursor-2": {Outcome: catalog.SyncOutcomeSuccess, LastSuccessAt: &now},
		},
		Models: []catalog.View{
			modelView("cursor-1", "composer-2.5", now, now, providers.StateHealthy),
			modelView("cursor-2", "composer-2.5", now, now, providers.StateHealthy),
		},
		AutoQuota: routing.AutoQuotaView{
			Pools: map[routing.AutoPoolID]routing.QuotaPressureState{
				routing.AccountPoolID("cursor-1"): {PoolID: routing.AccountPoolID("cursor-1"), Pressure: &highPressure, Weight: &lowWeight},
				routing.AccountPoolID("cursor-2"): {PoolID: routing.AccountPoolID("cursor-2"), Pressure: &lowPressure, Weight: &highWeight},
			},
		},
	}

	ctx := routing.RoutingContext{
		RequestID:     "req-multi-1",
		ExplicitModel: "composer-2.5",
		Structural:    routing.StructuralRequest{ModelPresent: true, ModelID: "composer-2.5"},
	}

	eligibility := routing.EvaluateEligibility(ctx, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)

	if decision.Rejection != "" {
		t.Fatalf("unexpected rejection: %s", decision.Rejection)
	}
	if decision.Selected == nil || decision.Selected.ProviderID != "cursor-2" {
		t.Fatalf("selected = %+v, want cursor-2 (lower pressure / higher weight)", decision.Selected)
	}
}

func TestFilterEligibleToAutoAllowlistForSnapshotMatchesByProviderType(t *testing.T) {
	t.Parallel()

	snapshot := routing.Snapshot{
		Admin: map[string]routing.AdminProviderFacts{
			"cursor-work": {ID: "cursor-work", Type: providers.TypeCursor, Known: true, Enabled: true, HasCredential: true},
		},
	}
	eligible := []routing.Target{
		{ProviderID: "cursor-work", ModelID: "composer-2.5"},
		{ProviderID: "cursor-work", ModelID: "unknown-model"},
	}

	filtered := routing.FilterEligibleToAutoAllowlistForSnapshot(eligible, snapshot)
	if len(filtered) != 1 || filtered[0].ProviderID != "cursor-work" || filtered[0].ModelID != "composer-2.5" {
		t.Fatalf("filtered = %+v, want [cursor-work/composer-2.5]", filtered)
	}
}
