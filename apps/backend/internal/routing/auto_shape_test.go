package routing_test

import (
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func viewWithContext(providerID, modelID string, contextLimit int64, caps map[string]catalog.CapabilityClaim, now time.Time) catalog.View {
	entry := catalog.ModelEntry{
		ProviderID:   providerID,
		ModelID:      modelID,
		Capabilities: caps,
		Limits:       catalog.Limits{Context: &contextLimit},
	}
	sync := catalog.ProviderSyncState{Outcome: catalog.SyncOutcomeSuccess, LastSuccessAt: &now}
	return catalog.DeriveView(entry, sync, providers.StateHealthy, now, 24*time.Hour)
}

func TestFilterAutoContextWindow_DropsSmallWindow(t *testing.T) {
	now := time.Now().UTC()
	smallCtx := int64(100)
	largeCtx := int64(1_000_000)
	snapshot := routing.Snapshot{
		Models: []catalog.View{
			viewWithContext(providers.OpenCodeGoID, "gpt-6-luna", smallCtx, nil, now),
			viewWithContext(providers.OpenCodeGoID, "glm-5.3-flash", largeCtx, nil, now),
		},
	}
	candidates := []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "glm-5.3-flash"},
	}
	understanding := routing.RequestUnderstanding{EstimatedInputTokens: 500}
	out, reasons := routing.FilterAutoContextWindow(candidates, understanding, snapshot)
	if len(out) != 1 || out[0].ModelID != "glm-5.3-flash" {
		t.Fatalf("out = %+v", out)
	}
	if len(reasons) == 0 || reasons[0] != "context_window_exceeded" {
		t.Fatalf("reasons = %v", reasons)
	}
}

func TestFilterAutoContextWindow_UnknownWindowKeeps(t *testing.T) {
	now := time.Now().UTC()
	entry := catalog.ModelEntry{
		ProviderID: providers.OpenCodeGoID,
		ModelID:    "gpt-6-luna",
		Capabilities: map[string]catalog.CapabilityClaim{
			"text_generation": {State: catalog.CapabilitySupported},
		},
	}
	sync := catalog.ProviderSyncState{Outcome: catalog.SyncOutcomeSuccess, LastSuccessAt: &now}
	snapshot := routing.Snapshot{
		Models: []catalog.View{catalog.DeriveView(entry, sync, providers.StateHealthy, now, 24*time.Hour)},
	}
	candidates := []routing.Target{{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"}}
	understanding := routing.RequestUnderstanding{EstimatedInputTokens: 500_000}
	out, _ := routing.FilterAutoContextWindow(candidates, understanding, snapshot)
	if len(out) != 1 {
		t.Fatalf("expected unknown context to keep candidate, got %+v", out)
	}
}

func TestSelectModelInPool_HighEffortPrefersReasoning(t *testing.T) {
	now := time.Now().UTC()
	snapshot := routing.Snapshot{
		Models: []catalog.View{
			viewWithContext(providers.OpenCodeGoID, "gpt-6-luna", 1_000_000, map[string]catalog.CapabilityClaim{
				"text_generation": {State: catalog.CapabilitySupported},
			}, now),
			viewWithContext(providers.OpenCodeGoID, "deepseek-v4.1-flash", 1_000_000, map[string]catalog.CapabilityClaim{
				"text_generation": {State: catalog.CapabilitySupported},
				"reasoning":       {State: catalog.CapabilitySupported},
			}, now),
		},
	}
	candidates := []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "deepseek-v4.1-flash"},
	}
	understanding := routing.RequestUnderstanding{
		ReasoningEffort:      routing.ReasoningEffortHigh,
		OptimizationProfile:  routing.ProfileBalanced,
		EstimatedInputTokens: 100,
	}
	selected, reasons := routing.SelectModelInPool(routing.PoolOpenCodeShared, candidates, snapshot, understanding)
	if selected.ModelID != "deepseek-v4.1-flash" {
		t.Fatalf("selected = %+v, want deepseek", selected)
	}
	if reasons[len(reasons)-1] != "reasoning_preferred" {
		t.Fatalf("reasons = %v", reasons)
	}
}

func TestSelectModelInPool_MediumEffortKeepsAdminOrder(t *testing.T) {
	now := time.Now().UTC()
	snapshot := routing.Snapshot{
		Models: []catalog.View{
			viewWithContext(providers.OpenCodeGoID, "gpt-6-luna", 1_000_000, map[string]catalog.CapabilityClaim{
				"text_generation": {State: catalog.CapabilitySupported},
			}, now),
			viewWithContext(providers.OpenCodeGoID, "deepseek-v4.1-flash", 1_000_000, map[string]catalog.CapabilityClaim{
				"text_generation": {State: catalog.CapabilitySupported},
				"reasoning":       {State: catalog.CapabilitySupported},
			}, now),
		},
	}
	candidates := []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "deepseek-v4.1-flash"},
	}
	understanding := routing.RequestUnderstanding{
		ReasoningEffort:      routing.ReasoningEffortMedium,
		OptimizationProfile:  routing.ProfileBalanced,
		EstimatedInputTokens: 100,
	}
	selected, reasons := routing.SelectModelInPool(routing.PoolOpenCodeShared, candidates, snapshot, understanding)
	if selected.ModelID != "gpt-6-luna" {
		t.Fatalf("selected = %+v, want gpt-6-luna", selected)
	}
	for _, r := range reasons {
		if r == "reasoning_preferred" || r == "reasoning_preference_unmatched" {
			t.Fatalf("unexpected heavy-shape reason %q", r)
		}
	}
}

func TestSelectModelInPool_HeavyShapeUnmatchedFallback(t *testing.T) {
	now := time.Now().UTC()
	snapshot := routing.Snapshot{
		Models: []catalog.View{
			viewWithContext(providers.OpenCodeGoID, "gpt-6-luna", 1_000_000, map[string]catalog.CapabilityClaim{
				"text_generation": {State: catalog.CapabilitySupported},
			}, now),
			viewWithContext(providers.OpenCodeGoID, "glm-5.3-flash", 1_000_000, map[string]catalog.CapabilityClaim{
				"text_generation": {State: catalog.CapabilitySupported},
			}, now),
		},
	}
	candidates := []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "glm-5.3-flash"},
	}
	understanding := routing.RequestUnderstanding{
		OptimizationProfile: routing.ProfileIntelligence,
		EstimatedInputTokens: 100,
	}
	selected, reasons := routing.SelectModelInPool(routing.PoolOpenCodeShared, candidates, snapshot, understanding)
	if selected.ModelID != "gpt-6-luna" {
		t.Fatalf("selected = %+v", selected)
	}
	found := false
	for _, r := range reasons {
		if r == "reasoning_preference_unmatched" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reasons = %v", reasons)
	}
}

func TestSelectModelInPool_EfficiencySkipsReasoningPreference(t *testing.T) {
	now := time.Now().UTC()
	snapshot := routing.Snapshot{
		Models: []catalog.View{
			viewWithContext(providers.OpenCodeGoID, "gpt-6-luna", 1_000_000, map[string]catalog.CapabilityClaim{
				"text_generation": {State: catalog.CapabilitySupported},
			}, now),
			viewWithContext(providers.OpenCodeGoID, "deepseek-v4.1-flash", 1_000_000, map[string]catalog.CapabilityClaim{
				"text_generation": {State: catalog.CapabilitySupported},
				"reasoning":       {State: catalog.CapabilitySupported},
			}, now),
		},
	}
	candidates := []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenCodeGoID, ModelID: "deepseek-v4.1-flash"},
	}
	understanding := routing.RequestUnderstanding{
		OptimizationProfile:  routing.ProfileEfficiency,
		ReasoningEffort:      routing.ReasoningEffortHigh,
		EstimatedInputTokens: 100,
	}
	selected, reasons := routing.SelectModelInPool(routing.PoolOpenCodeShared, candidates, snapshot, understanding)
	if selected.ModelID != "gpt-6-luna" {
		t.Fatalf("selected = %+v, want gpt-6-luna (no reasoning steer on efficiency)", selected)
	}
	for _, r := range reasons {
		if r == "reasoning_preferred" || r == "reasoning_preference_unmatched" {
			t.Fatalf("efficiency must not apply reasoning preference, got %q", r)
		}
	}
}

