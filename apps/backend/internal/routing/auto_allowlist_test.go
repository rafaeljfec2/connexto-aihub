package routing_test

import (
	"testing"

	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestAutoAllowlist_CountAndMembership(t *testing.T) {
	if len(routing.AutoMVPAllowlist) != 7 {
		t.Fatalf("allowlist len = %d, want 7", len(routing.AutoMVPAllowlist))
	}
	unknown := routing.Target{ProviderID: providers.OpenRouterID, ModelID: "not-in-list"}
	if routing.IsAutoAllowlistTarget(unknown) {
		t.Fatal("unexpected allowlist match for external model")
	}
}

func TestFilterEligibleToAutoAllowlist(t *testing.T) {
	eligible := []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenRouterID, ModelID: "anthropic/claude-sonnet-4.5"},
	}
	filtered := routing.FilterEligibleToAutoAllowlist(eligible)
	if len(filtered) != 1 || filtered[0].ModelID != "gpt-6-luna" {
		t.Fatalf("filtered = %+v", filtered)
	}
}
