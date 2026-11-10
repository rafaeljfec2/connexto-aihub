package routing_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func modelView(providerID, modelID string, now, success time.Time, health providers.State) catalog.View {
	entry := catalog.ModelEntry{
		ProviderID: providerID,
		ModelID:    modelID,
		Capabilities: map[string]catalog.CapabilityClaim{
			"text_generation": {State: catalog.CapabilitySupported},
		},
	}

	sync := catalog.ProviderSyncState{LastSuccessAt: &success, Outcome: catalog.SyncOutcomeSuccess}

	return catalog.DeriveView(entry, sync, health, now, 24*time.Hour)
}

func assertReason(t *testing.T, reasons []routing.ReasonCode, want routing.ReasonCode) {
	t.Helper()
	for _, reason := range reasons {
		if reason == want {
			return
		}
	}
	t.Fatalf("reasons = %v, want %q", reasons, want)
}

