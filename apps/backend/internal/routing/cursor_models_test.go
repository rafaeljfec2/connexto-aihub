package routing_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestSelectDeterministicComposerInfersCursorProvider(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	snapshot.Admin[providers.CursorID] = routing.AdminProviderFacts{
		ID: providers.CursorID, Type: providers.TypeCursor, Known: true, Enabled: true, HasCredential: true,
	}
	snapshot.Runtime[providers.CursorID] = routing.RuntimeProviderFacts{
		Bound: true, ResponsesSupported: true, RuntimeEnabled: true,
	}
	snapshot.Sync[providers.CursorID] = catalog.ProviderSyncState{
		LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess,
	}
	snapshot.Models = append(snapshot.Models, modelView(providers.CursorID, "composer-2.5", now, success, providers.StateHealthy))

	ctx := routing.RoutingContext{
		RequestID:  "req-composer",
		Structural: routing.StructuralRequest{ModelPresent: true, ModelID: "composer-2.5"},
		Understanding: routing.RequestUnderstanding{
			Requirements: routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}},
		},
	}
	eligibility := routing.EligibilityResult{
		Eligible: []routing.Target{{ProviderID: providers.CursorID, ModelID: "composer-2.5"}},
		Coverage: routing.CoverageEvidence{CompleteFreshCoverageForEligible: true},
	}
	decision := routing.SelectDeterministic(ctx, snapshot, eligibility)
	assertSelected(t, decision, providers.CursorID, "composer-2.5", routing.OriginExplicitProviderAndModel)
}
