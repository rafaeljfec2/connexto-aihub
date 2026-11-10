package routing

import (
	"aihub/internal/providers"
)

// DefaultVirtualPriority defines the administrative priority ranking for virtual model resolution.
var DefaultVirtualPriority = []Target{
	{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
	{ProviderID: providers.OpenRouterID, ModelID: "anthropic/claude-haiku-4.5"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "glm-5.3-flash"},
	{ProviderID: providers.OpenCodeGoID, ModelID: "claude-haiku-5-5"},
}

// selectVirtualAuto resolves a virtual model intent (e.g. aihub/auto) into a concrete eligible target.
func selectVirtualAuto(decision RoutingDecision, snapshot Snapshot, eligibility EligibilityResult) RoutingDecision {
	switch len(eligibility.Eligible) {
	case 0:
		return reject(decision, classifyZeroEligible(eligibility.Evaluated), nil)
	case 1:
		target := eligibility.Eligible[0]
		decision.Selected = &target
		decision.Origin = OriginVirtualAuto
		decision.Criteria = []string{"rule:virtual_auto_unique_candidate", "profile:balanced"}
		return decision
	default:
		// Multiple eligible candidates: rank using administrative priority
		for _, priorityTarget := range DefaultVirtualPriority {
			if containsTarget(eligibility.Eligible, priorityTarget) {
				selected := priorityTarget
				decision.Selected = &selected
				decision.Origin = OriginVirtualAuto
				decision.Criteria = []string{"rule:virtual_auto_ranked_priority", "profile:balanced"}
				return decision
			}
		}

		// Deterministic fallback: choose the first candidate from the ordered eligibility list
		target := eligibility.Eligible[0]
		decision.Selected = &target
		decision.Origin = OriginVirtualAuto
		decision.Criteria = []string{"rule:virtual_auto_snapshot_fallback", "profile:balanced"}
		return decision
	}
}

