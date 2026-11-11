package routing_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestEvaluateEligibilityDoesNotSelectAmongMultipleEligible(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	success := now.Add(-time.Hour)

	snapshot := routing.Snapshot{
		CapturedAt: now,
		StaleAfter: 24 * time.Hour,
		Admin: map[string]routing.AdminProviderFacts{
			"opencode-go": {ID: "opencode-go", Known: true, Enabled: true, HasCredential: true},
		},
		Runtime: map[string]routing.RuntimeProviderFacts{
			"opencode-go": {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateUnknown},
		},
		Sync: map[string]catalog.ProviderSyncState{
			"opencode-go": {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
		},
		Models: []catalog.View{
			modelView("opencode-go", "model-a", now, success, providers.StateUnknown),
			modelView("opencode-go", "model-b", now, success, providers.StateUnknown),
		},
	}

	result := routing.EvaluateEligibility(routing.RoutingContext{
		RequestID: "req-1",
		Structural: routing.StructuralRequest{},
	}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})

	if len(result.Eligible) != 2 {
		t.Fatalf("eligible count = %d, want 2", len(result.Eligible))
	}

	if result.Eligible[0].ModelID == result.Eligible[1].ModelID {
		t.Fatal("expected two distinct eligible targets")
	}
}

func TestEvaluateEligibilityNeverSyncedUsesCatalogStale(t *testing.T) {
	now := time.Now().UTC()
	snapshot := baseSnapshot(now, now)
	snapshot.Sync["opencode-go"] = catalog.ProviderSyncState{Outcome: catalog.SyncOutcomeNever}
	snapshot.Models[0] = catalog.DeriveView(
		snapshot.Models[0].Entry,
		catalog.ProviderSyncState{Outcome: catalog.SyncOutcomeNever},
		providers.StateHealthy,
		now,
		24*time.Hour,
	)

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonCatalogStale)
	for _, reason := range result.Evaluated[0].Reasons {
		if reason == routing.ReasonModelNotFound {
			t.Fatal("never-synced catalog must not use model_not_found")
		}
	}
}

func TestEvaluateEligibilityBoundRuntimeRequiresAdminRecord(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	delete(snapshot.Admin, "opencode-go")

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonCredentialMissing)
	if result.Evaluated[0].ExclusionClass != routing.ExclusionInsufficient {
		t.Fatalf("class = %q, want insufficient", result.Evaluated[0].ExclusionClass)
	}
}

func TestEvaluateEligibilityProviderUnknown(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	delete(snapshot.Admin, "opencode-go")
	delete(snapshot.Runtime, "opencode-go")

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonProviderUnknown)
}

func TestEvaluateEligibilityResponsesUnsupported(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	snapshot.Runtime["opencode-go"] = routing.RuntimeProviderFacts{Bound: true, ResponsesSupported: false, RuntimeEnabled: true}

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonResponsesUnsupported)
}

func TestEvaluateEligibilityRuntimeUnavailable(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	snapshot.Runtime["opencode-go"] = routing.RuntimeProviderFacts{Bound: true, ResponsesSupported: true, RuntimeEnabled: false}

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonProviderUnavailable)
}

func TestEvaluateEligibilityCapabilityUnsupported(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	entry := snapshot.Models[0].Entry
	entry.Capabilities["reasoning"] = catalog.CapabilityClaim{State: catalog.CapabilityUnsupported}
	snapshot.Models[0].Entry = entry

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"reasoning"}})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonCapabilityUnsupported)
}

func TestEvaluateEligibilityLimitUnknown(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	requested := int64(100)
	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{MaxOutputTokens: &requested, RequiredCapabilities: []string{"text_generation"}})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonLimitUnknown)
}

func TestEvaluateEligibilityNoEligibleCandidates(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	snapshot.Admin["opencode-go"] = routing.AdminProviderFacts{ID: "opencode-go", Known: true, Enabled: false, HasCredential: true}

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{})
	if len(result.Eligible) != 0 {
		t.Fatal("expected zero eligible candidates")
	}
}

func TestEvaluateEligibilityProviderDisabled(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)

	snapshot := baseSnapshot(now, success)
	snapshot.Admin["opencode-go"] = routing.AdminProviderFacts{ID: "opencode-go", Known: true, Enabled: false, HasCredential: true}

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{})
	if len(result.Eligible) != 0 {
		t.Fatalf("eligible = %+v, want none", result.Eligible)
	}

	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonProviderDisabled)
}

func TestEvaluateEligibilityCredentialMissing(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	snapshot.Admin["opencode-go"] = routing.AdminProviderFacts{ID: "opencode-go", Known: true, Enabled: true, HasCredential: false}

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonCredentialMissing)
}

func TestEvaluateEligibilityHealthDoesNotBlockEligible(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	snapshot.Runtime["opencode-go"] = routing.RuntimeProviderFacts{
		Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateUnhealthy,
	}

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	if len(result.Eligible) != 1 {
		t.Fatalf("eligible count = %d, want 1 despite unhealthy health context", len(result.Eligible))
	}
}

func TestEvaluateEligibilityCatalogStale(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	staleSuccess := now.Add(-48 * time.Hour)

	snapshot := baseSnapshot(now, staleSuccess)
	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"text_generation"}})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonCatalogStale)
}

func TestEvaluateEligibilityCapabilityUnknownIsNotCompatible(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	entry := snapshot.Models[0].Entry
	entry.Capabilities["reasoning"] = catalog.CapabilityClaim{State: catalog.CapabilityUnknown}
	snapshot.Models[0].Entry = entry

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{RequiredCapabilities: []string{"reasoning"}})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonCapabilityUnknown)
	if result.Evaluated[0].Eligible {
		t.Fatal("unknown capability must not be eligible")
	}
}

func TestEvaluateEligibilityRequirementUnmapped(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{Unmapped: true})
	if !result.RequirementUnmapped {
		t.Fatal("expected requirement_unmapped flag")
	}

	if len(result.Eligible) != 0 {
		t.Fatal("unmapped requirement must not yield eligible candidates")
	}
}

func TestEvaluateEligibilityLimitExceeded(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	limit := int64(100)
	entry := snapshot.Models[0].Entry
	entry.Limits.Output = &limit
	snapshot.Models[0].Entry = entry

	requested := int64(200)
	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{MaxOutputTokens: &requested, RequiredCapabilities: []string{"text_generation"}})
	assertReason(t, result.Evaluated[0].Reasons, routing.ReasonLimitExceeded)
}

func TestCoverageIncompleteWhenProviderEligibleButCatalogStale(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	staleSuccess := now.Add(-48 * time.Hour)
	snapshot := baseSnapshot(now, staleSuccess)

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{})
	if result.Coverage.CompleteFreshCoverageForEligible {
		t.Fatal("expected incomplete fresh coverage")
	}
}

func TestCoverageIgnoresTransportOnlyProviderWithoutCatalog(t *testing.T) {
	now := time.Now().UTC()
	success := now.Add(-time.Hour)
	snapshot := baseSnapshot(now, success)
	snapshot.Admin[providers.ConnextoProxyID] = routing.AdminProviderFacts{
		ID: providers.ConnextoProxyID, Known: true, Enabled: true, HasCredential: true,
	}
	snapshot.Runtime[providers.ConnextoProxyID] = routing.RuntimeProviderFacts{
		Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy,
	}

	result := routing.EvaluateEligibility(routing.RoutingContext{RequestID: "req"}, snapshot, routing.RequirementStatus{})
	if !result.Coverage.CompleteFreshCoverageForEligible {
		t.Fatal("transport-only catalog provider must not require fresh catalog sync for coverage")
	}
}

func baseSnapshot(now, success time.Time) routing.Snapshot {
	return routing.Snapshot{
		CapturedAt: now,
		StaleAfter: 24 * time.Hour,
		Admin: map[string]routing.AdminProviderFacts{
			"opencode-go": {ID: "opencode-go", Known: true, Enabled: true, HasCredential: true},
		},
		Runtime: map[string]routing.RuntimeProviderFacts{
			"opencode-go": {Bound: true, ResponsesSupported: true, RuntimeEnabled: true, Health: providers.StateHealthy},
		},
		Sync: map[string]catalog.ProviderSyncState{
			"opencode-go": {LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess},
		},
		Models: []catalog.View{
			modelView("opencode-go", "model-a", now, success, providers.StateHealthy),
		},
	}
}
