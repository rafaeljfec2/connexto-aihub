package routing

import (
	"strings"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

// SelectDeterministic applies contract §4 precedence to an eligibility result.
// It never uses lexical order as a tie-break among multiple eligible candidates.
func SelectDeterministic(ctx RoutingContext, snapshot Snapshot, eligibility EligibilityResult) RoutingDecision {
	decision := RoutingDecision{
		RequestID:   ctx.RequestID,
		Eligibility: eligibility,
	}

	if eligibility.RequirementUnmapped {
		return reject(decision, RejectRequirementUnmapped, nil)
	}

	explicitProvider := strings.TrimSpace(ctx.ExplicitProvider)
	hasExplicitProvider := explicitProvider != ""
	explicitModelID := explicitModelID(ctx)

	if hasExplicitProvider && !providerKnown(snapshot, explicitProvider) {
		return reject(decision, RejectProviderNotFound, nil)
	}

	switch {
	case hasExplicitProvider && explicitModelID != "":
		return selectExplicitProviderAndModel(decision, snapshot, eligibility, explicitProvider, explicitModelID)
	case hasExplicitProvider:
		return selectExplicitProviderOnly(decision, snapshot, eligibility, explicitProvider)
	case explicitModelID != "":
		return selectExplicitModelOnly(decision, snapshot, eligibility, explicitModelID)
	default:
		return selectCompatibilityDefault(decision, snapshot, eligibility)
	}
}

func explicitModelID(ctx RoutingContext) string {
	if ctx.Structural.ModelPresent {
		return ctx.Structural.ModelID
	}

	if strings.TrimSpace(ctx.ExplicitModel) != "" {
		return ctx.ExplicitModel
	}

	return ""
}

func selectExplicitProviderAndModel(
	decision RoutingDecision,
	snapshot Snapshot,
	eligibility EligibilityResult,
	providerID, modelID string,
) RoutingDecision {
	if modelPresentUnderOtherProvider(snapshot, providerID, modelID) && !modelPresentOnProvider(snapshot, providerID, modelID) {
		return reject(decision, RejectProviderModelConflict, nil)
	}

	target := Target{ProviderID: providerID, ModelID: modelID}
	if containsTarget(eligibility.Eligible, target) {
		decision.Selected = &target
		decision.Origin = OriginExplicitProviderAndModel
		decision.Criteria = []string{"rule:explicit_provider_and_model"}
		return decision
	}

	scope := evaluatedForProvider(eligibility.Evaluated, providerID)
	if len(scope) == 0 && !modelPresentOnProvider(snapshot, providerID, modelID) {
		if eligibility.Coverage.CompleteFreshCoverageForEligible {
			return reject(decision, RejectModelNotFound, nil)
		}

		return reject(decision, RejectRoutingUnavailable, nil)
	}

	return reject(decision, classifyZeroEligible(scope), nil)
}

func selectExplicitProviderOnly(
	decision RoutingDecision,
	snapshot Snapshot,
	eligibility EligibilityResult,
	providerID string,
) RoutingDecision {
	candidates := filterEligibleByProvider(eligibility.Eligible, providerID)

	switch len(candidates) {
	case 1:
		target := candidates[0]
		decision.Selected = &target
		decision.Origin = OriginExplicitProvider
		decision.Criteria = []string{"rule:explicit_provider_unique_model"}
		return decision
	case 0:
		scope := evaluatedForProvider(eligibility.Evaluated, providerID)
		if len(scope) == 0 && !coverageCatalogFreshForProvider(snapshot, eligibility, providerID) {
			return reject(decision, RejectRoutingUnavailable, nil)
		}

		return reject(decision, classifyZeroEligible(scope), nil)
	default:
		if !coverageCatalogFreshForProvider(snapshot, eligibility, providerID) {
			return reject(decision, RejectRoutingUnavailable, nil)
		}

		return reject(decision, RejectRoutingAmbiguous, nil)
	}
}

func selectExplicitModelOnly(
	decision RoutingDecision,
	snapshot Snapshot,
	eligibility EligibilityResult,
	modelID string,
) RoutingDecision {
	matches := filterEligibleByModel(eligibility.Eligible, modelID)

	switch len(matches) {
	case 1:
		if !eligibility.Coverage.CompleteFreshCoverageForEligible {
			return reject(decision, RejectRoutingUnavailable, nil)
		}

		target := matches[0]
		decision.Selected = &target
		decision.Origin = OriginExplicitModel
		decision.Criteria = []string{"rule:explicit_model_unique_pair"}
		return decision
	case 0:
		if !eligibility.Coverage.CompleteFreshCoverageForEligible {
			return reject(decision, RejectRoutingUnavailable, nil)
		}

		if !modelPresentOnRelevantProviders(snapshot, modelID) {
			return reject(decision, RejectModelNotFound, nil)
		}

		scope := evaluatedForModel(eligibility.Evaluated, modelID)
		return reject(decision, classifyZeroEligible(scope), nil)
	default:
		if !eligibility.Coverage.CompleteFreshCoverageForEligible {
			return reject(decision, RejectRoutingUnavailable, nil)
		}

		return reject(decision, RejectRoutingAmbiguous, nil)
	}
}

func selectCompatibilityDefault(decision RoutingDecision, snapshot Snapshot, eligibility EligibilityResult) RoutingDecision {
	openCodeEligible := filterEligibleByProvider(eligibility.Eligible, providers.OpenCodeGoID)

	switch len(openCodeEligible) {
	case 1:
		target := openCodeEligible[0]
		decision.Selected = &target
		decision.Origin = OriginCompatibilityDefault
		decision.Criteria = []string{"rule:compatibility_default_opencode_go"}
		return decision
	case 0:
		if !coverageCatalogFreshForProvider(snapshot, eligibility, providers.OpenCodeGoID) || !eligibility.Coverage.CompleteFreshCoverageForEligible {
			return reject(decision, RejectRoutingUnavailable, nil)
		}

		if len(eligibility.Eligible) == 1 {
			target := eligibility.Eligible[0]
			decision.Selected = &target
			decision.Origin = OriginAutomaticUniqueCandidate
			decision.Criteria = []string{"rule:automatic_unique_candidate"}
			return decision
		}

		if len(eligibility.Eligible) == 0 {
			return reject(decision, classifyZeroEligible(eligibility.Evaluated), nil)
		}

		return reject(decision, RejectRoutingAmbiguous, nil)
	default:
		if !coverageCatalogFreshForProvider(snapshot, eligibility, providers.OpenCodeGoID) {
			return reject(decision, RejectRoutingUnavailable, nil)
		}

		return reject(decision, RejectRoutingAmbiguous, nil)
	}
}

func reject(decision RoutingDecision, code RejectionCode, criteria []string) RoutingDecision {
	decision.Rejection = code
	if len(criteria) == 0 {
		criteria = []string{"rejection:" + string(code)}
	}
	decision.Criteria = criteria
	return decision
}

func classifyZeroEligible(evaluated []CandidateEvaluation) RejectionCode {
	if len(evaluated) == 0 {
		return RejectRoutingUnavailable
	}

	hasInsufficient := false
	hasIncompatible := false

	for _, candidate := range evaluated {
		if candidate.Eligible {
			continue
		}

		switch candidate.ExclusionClass {
		case ExclusionInsufficient:
			hasInsufficient = true
		case ExclusionIncompatible:
			hasIncompatible = true
		}
	}

	if hasInsufficient {
		return RejectRoutingUnavailable
	}

	if hasIncompatible {
		return RejectNoEligibleModel
	}

	return RejectRoutingUnavailable
}

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

func modelPresentOnProvider(snapshot Snapshot, providerID, modelID string) bool {
	for _, view := range snapshot.Models {
		if view.Entry.ProviderID != providerID || view.Entry.ModelID != modelID {
			continue
		}

		return view.Presence != catalog.PresenceMissing
	}

	return false
}

func modelPresentUnderOtherProvider(snapshot Snapshot, explicitProviderID, modelID string) bool {
	for _, view := range snapshot.Models {
		if view.Entry.ProviderID == explicitProviderID || view.Entry.ModelID != modelID {
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
		if view.Entry.ModelID != modelID || view.Presence == catalog.PresenceMissing {
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
		if target.ModelID == modelID {
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
		if candidate.Target.ModelID == modelID {
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}
