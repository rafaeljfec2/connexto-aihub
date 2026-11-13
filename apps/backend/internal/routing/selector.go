package routing

import (
	"strings"

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

	if !hasExplicitProvider && strings.HasPrefix(explicitModelID, "antigravity:") {
		explicitProvider = firstAntigravityProviderID(snapshot)
		if explicitProvider != "" {
			hasExplicitProvider = true
		}
	} else if !hasExplicitProvider && IsCursorProxyModelID(explicitModelID) {
		explicitProvider = firstCursorProviderID(snapshot)
		if explicitProvider != "" {
			hasExplicitProvider = true
		}
	} else if hasExplicitProvider && providerTypeOf(snapshot, explicitProvider) == providers.TypeAntigravity {
		explicitModelID = resolveAntigravityModelID(snapshot, explicitProvider, explicitModelID)
	}

	if hasExplicitProvider && !providerKnown(snapshot, explicitProvider) {
		return reject(decision, RejectProviderNotFound, nil)
	}

	switch {
	case IsVirtualModel(explicitModelID):
		return selectVirtualAuto(decision, ctx, snapshot, eligibility)
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
	if providerTypeOf(snapshot, providerID) == providers.TypeAntigravity && !strings.HasPrefix(modelID, "antigravity:") {
		alt := Target{ProviderID: providerID, ModelID: "antigravity:" + modelID}
		if containsTarget(eligibility.Eligible, alt) {
			decision.Selected = &alt
			decision.Origin = OriginExplicitProviderAndModel
			decision.Criteria = []string{"rule:explicit_provider_and_model"}
			return decision
		}
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
		if sameProviderType(snapshot, matches) {
			return selectSameProviderAccountAWRR(decision, snapshot, matches)
		}

		return reject(decision, RejectRoutingAmbiguous, nil)
	}
}

func firstAntigravityProviderID(snapshot Snapshot) string {
	var count int
	var only string
	for id := range snapshot.Admin {
		if providerTypeOf(snapshot, id) == providers.TypeAntigravity {
			count++
			only = id
		}
	}
	if count == 1 {
		return only
	}
	if count > 1 {
		return ""
	}
	return providers.AntigravityID
}

func sameProviderType(snapshot Snapshot, matches []Target) bool {
	if len(matches) < 2 {
		return false
	}
	firstType := providerTypeOf(snapshot, matches[0].ProviderID)
	for _, m := range matches[1:] {
		if providerTypeOf(snapshot, m.ProviderID) != firstType {
			return false
		}
	}
	return true
}

func selectSameProviderAccountAWRR(decision RoutingDecision, snapshot Snapshot, matches []Target) RoutingDecision {
	resolver := func(t Target) AutoPoolID {
		return AccountPoolID(t.ProviderID)
	}
	poolID, poolReasons := SelectPoolAWRRWithResolver(matches, snapshot.AutoQuota, resolver)
	if poolID == "" {
		return reject(decision, RejectRoutingAmbiguous, poolReasons)
	}
	for i := range matches {
		if resolver(matches[i]) == poolID {
			target := matches[i]
			ProvisionalDebit(poolID)
			decision.Selected = &target
			decision.Origin = OriginExplicitModel
			decision.SelectedPool = string(poolID)
			decision.RoutingReasonCode = "awrr_account_tiebreak"
			decision.Criteria = append([]string{"rule:explicit_model_awrr_tiebreak", "pool:" + string(poolID)}, poolReasons...)
			return decision
		}
	}
	return reject(decision, RejectRoutingAmbiguous, poolReasons)
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
