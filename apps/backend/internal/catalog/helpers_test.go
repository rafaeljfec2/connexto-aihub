package catalog_test

import (
	"encoding/json"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

func testNow() time.Time {
	return time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
}

func testBoolPtr(value bool) *bool {
	return &value
}

func testInt64Ptr(value int64) *int64 {
	return &value
}

func testTimePtr(value time.Time) *time.Time {
	return &value
}

func testDescriptor() providers.ModelDescriptor {
	return providers.ModelDescriptor{
		ID:                      "deepseek-v4.1-flash",
		CanonicalID:             "deepseek/deepseek-v4.1-flash",
		Name:                    "DeepSeek V4.1 Flash",
		Type:                    "chat",
		Family:                  "deepseek-flash",
		InputModalities:         []string{"text", "image/png", "audio/wav"},
		InputModalitiesPresent:  true,
		OutputModalities:        []string{"text", "audio/wav"},
		OutputModalitiesPresent: true,
		LimitContext:            testInt64Ptr(1000000),
		LimitInput:              testInt64Ptr(900000),
		LimitOutput:             testInt64Ptr(384000),
		ToolsSupported:          testBoolPtr(true),
		ReasoningSupported:      testBoolPtr(true),
		StructuredOutput:        testBoolPtr(true),
		Raw:                     json.RawMessage(`{"id":"deepseek-v4.1-flash"}`),
	}
}

func testClaim(state catalog.CapabilityState, confidence catalog.CapabilityConfidence) catalog.CapabilityClaim {
	return catalog.CapabilityClaim{
		State:        state,
		Provenance:   catalog.ProvenanceProviderCatalog,
		Confidence:   confidence,
		Verification: catalog.VerificationUnverified,
	}
}

func assertClaim(t *testing.T, capabilities map[string]catalog.CapabilityClaim, name string, want catalog.CapabilityClaim) {
	t.Helper()

	got, ok := capabilities[name]
	if !ok {
		t.Fatalf("capability %q missing from entry", name)
	}

	if got != want {
		t.Fatalf("capability %q = %+v, want %+v", name, got, want)
	}
}

func testEntry() catalog.ModelEntry {
	return catalog.NormalizeEntry(providers.OpenCodeGoID, testDescriptor(), testNow())
}

func testSyncState(lastSuccess *time.Time, outcome catalog.SyncOutcome) catalog.ProviderSyncState {
	return catalog.ProviderSyncState{
		LastSuccessAt: lastSuccess,
		Outcome:       outcome,
	}
}
