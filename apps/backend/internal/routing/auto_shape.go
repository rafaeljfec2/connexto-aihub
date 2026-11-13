package routing

import (
	"aihub/internal/catalog"
	"aihub/internal/providers"
)

// FilterAutoAntigravityWithoutTools drops Antigravity when the request needs
// executable tools. Connexto Proxy rejects OpenAI tools for Antigravity even when
// the catalog still claims function_tools=supported.
func FilterAutoAntigravityWithoutTools(candidates []Target, understanding RequestUnderstanding) ([]Target, []string) {
	needsTools := false
	for _, capName := range understanding.Requirements.RequiredCapabilities {
		if capName == "function_tools" || capName == "custom_tools" {
			needsTools = true
			break
		}
	}
	if !needsTools || len(candidates) == 0 {
		return candidates, nil
	}
	out := make([]Target, 0, len(candidates))
	dropped := false
	for _, t := range candidates {
		if t.ProviderID == providers.AntigravityID || len(t.ModelID) >= 12 && t.ModelID[:12] == "antigravity:" {
			dropped = true
			continue
		}
		out = append(out, t)
	}
	if !dropped {
		return candidates, nil
	}
	return out, []string{"antigravity_tools_unsupported"}
}

// FilterAutoContextWindow drops allowlist candidates whose known context window
// is smaller than the structural input token estimate (aihub/auto path only).
func FilterAutoContextWindow(candidates []Target, understanding RequestUnderstanding, snapshot Snapshot) ([]Target, []string) {
	if len(candidates) == 0 {
		return nil, nil
	}
	estimate := int64(understanding.EstimatedInputTokens)
	if estimate <= 0 {
		return candidates, nil
	}
	out := make([]Target, 0, len(candidates))
	var reasons []string
	seen := make(map[string]struct{}, 2)
	for _, t := range candidates {
		view, ok := catalogViewForTarget(snapshot, t)
		if !ok || view.Entry.Limits.Context == nil {
			out = append(out, t)
			continue
		}
		if estimate > *view.Entry.Limits.Context {
			if _, dup := seen["context_window_exceeded"]; !dup {
				seen["context_window_exceeded"] = struct{}{}
				reasons = append(reasons, "context_window_exceeded")
			}
			continue
		}
		out = append(out, t)
	}
	return out, reasons
}

func isHeavyRequestShape(u RequestUnderstanding) bool {
	if u.OptimizationProfile == ProfileIntelligence {
		return true
	}
	switch u.ReasoningEffort {
	case ReasoningEffortHigh, ReasoningEffortXHigh:
		return true
	}
	if u.EstimatedInputTokens >= 30000 {
		return true
	}
	if u.InputItemCount >= 8 {
		return true
	}
	return false
}

func modelHasReasoningSupported(snapshot Snapshot, t Target) bool {
	view, ok := catalogViewForTarget(snapshot, t)
	if !ok {
		return false
	}
	claim, exists := view.Entry.Capabilities["reasoning"]
	return exists && claim.State == catalog.CapabilitySupported
}

func preferReasoningInAdminOrder(ordered []Target, snapshot Snapshot) (Target, bool) {
	for _, t := range ordered {
		if modelHasReasoningSupported(snapshot, t) {
			return t, true
		}
	}
	return Target{}, false
}

func catalogViewForTarget(snapshot Snapshot, t Target) (catalog.View, bool) {
	for _, view := range snapshot.Models {
		if view.Entry.ModelID != t.ModelID {
			continue
		}
		if view.Entry.ProviderID == t.ProviderID {
			return view, true
		}
		if providerTypeOf(snapshot, view.Entry.ProviderID) == t.ProviderID {
			return view, true
		}
	}
	return catalog.View{}, false
}
