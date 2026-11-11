package routing

import (
	"strings"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

func providerKnown(snapshot Snapshot, providerID string) bool {
	admin, adminKnown := snapshot.Admin[providerID]
	_, runtimeKnown := snapshot.Runtime[providerID]

	if !adminKnown && !runtimeKnown {
		return false
	}

	if adminKnown && !admin.Known {
		return false
	}

	return true
}

func coverageCatalogFreshForProvider(snapshot Snapshot, eligibility EligibilityResult, providerID string) bool {
	for _, coverage := range eligibility.Coverage.Providers {
		if coverage.ProviderID != providerID {
			continue
		}

		if !coverage.PassesProviderChecks {
			return true
		}

		return coverage.CatalogFreshAccepted
	}

	reasons, _ := evaluateProvider(providerID, snapshot)
	if len(reasons) > 0 {
		return true
	}

	sync, ok := snapshot.Sync[providerID]
	if !ok {
		return false
	}

	return providerCatalogFresh(sync, snapshot.CapturedAt, snapshot.StaleAfter)
}

func resolveAntigravityModelID(snapshot Snapshot, providerID, modelID string) string {
	if modelID == "" {
		return ""
	}
	if modelPresentOnProvider(snapshot, providerID, modelID) {
		return modelID
	}
	if (providerID == providers.AntigravityID || providerID == "") && !strings.HasPrefix(modelID, "antigravity:") {
		prefixed := "antigravity:" + modelID
		if modelPresentOnProvider(snapshot, providers.AntigravityID, prefixed) {
			return prefixed
		}
	}
	return modelID
}

func modelPresentOnProvider(snapshot Snapshot, providerID, modelID string) bool {
	for _, view := range snapshot.Models {
		if view.Entry.ProviderID != providerID {
			continue
		}
		if view.Entry.ModelID == modelID || (providerID == providers.AntigravityID && view.Entry.ModelID == "antigravity:"+modelID) {
			return view.Presence != catalog.PresenceMissing
		}
	}

	return false
}

func modelPresentUnderOtherProvider(snapshot Snapshot, explicitProviderID, modelID string) bool {
	for _, view := range snapshot.Models {
		if view.Entry.ProviderID == explicitProviderID {
			continue
		}
		matches := view.Entry.ModelID == modelID || (view.Entry.ProviderID == providers.AntigravityID && view.Entry.ModelID == "antigravity:"+modelID)
		if !matches {
			continue
		}

		if view.Presence == catalog.PresenceMissing {
			continue
		}

		if !providerPassingChecks(snapshot, view.Entry.ProviderID) {
			continue
		}

		return true
	}

	return false
}

func modelPresentOnRelevantProviders(snapshot Snapshot, modelID string) bool {
	for _, view := range snapshot.Models {
		matches := view.Entry.ModelID == modelID || (view.Entry.ProviderID == providers.AntigravityID && view.Entry.ModelID == "antigravity:"+modelID)
		if !matches || view.Presence == catalog.PresenceMissing {
			continue
		}

		if !providerPassingChecks(snapshot, view.Entry.ProviderID) {
			continue
		}

		return true
	}

	return false
}

func providerPassingChecks(snapshot Snapshot, providerID string) bool {
	reasons, _ := evaluateProvider(providerID, snapshot)
	return len(reasons) == 0
}

func containsTarget(targets []Target, want Target) bool {
	for _, target := range targets {
		if target == want {
			return true
		}
	}

	return false
}

func filterEligibleByProvider(targets []Target, providerID string) []Target {
	filtered := make([]Target, 0, len(targets))
	for _, target := range targets {
		if target.ProviderID == providerID {
			filtered = append(filtered, target)
		}
	}

	return filtered
}

func filterEligibleByModel(targets []Target, modelID string) []Target {
	filtered := make([]Target, 0, len(targets))
	for _, target := range targets {
		if target.ModelID == modelID || (target.ProviderID == providers.AntigravityID && target.ModelID == "antigravity:"+modelID) {
			filtered = append(filtered, target)
		}
	}

	return filtered
}

func evaluatedForProvider(evaluated []CandidateEvaluation, providerID string) []CandidateEvaluation {
	filtered := make([]CandidateEvaluation, 0, len(evaluated))
	for _, candidate := range evaluated {
		if candidate.Target.ProviderID == providerID {
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}

func evaluatedForModel(evaluated []CandidateEvaluation, modelID string) []CandidateEvaluation {
	filtered := make([]CandidateEvaluation, 0, len(evaluated))
	for _, candidate := range evaluated {
		if candidate.Target.ModelID == modelID || (candidate.Target.ProviderID == providers.AntigravityID && candidate.Target.ModelID == "antigravity:"+modelID) {
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}

