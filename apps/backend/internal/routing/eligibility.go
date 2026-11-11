package routing

import (
	"sort"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

// EvaluateEligibility applies the E08 routing contract to a snapshot and request context.
// It never selects a single winner when multiple candidates remain eligible.
func EvaluateEligibility(ctx RoutingContext, snapshot Snapshot, requirements RequirementStatus) EligibilityResult {
	result := EligibilityResult{
		RequestID:    ctx.RequestID,
		Requirements: requirements,
	}

	if requirements.Unmapped {
		result.RequirementUnmapped = true
		result.Coverage = buildCoverage(snapshot)
		return result
	}

	if requirements.MaxOutputTokens == nil && ctx.Structural.MaxOutputTokens != nil {
		requirements.MaxOutputTokens = ctx.Structural.MaxOutputTokens
		result.Requirements = requirements
	}

	evaluated := make([]CandidateEvaluation, 0, len(snapshot.Models))

	for _, view := range snapshot.Models {
		target := Target{ProviderID: view.Entry.ProviderID, ModelID: view.Entry.ModelID}
		reasons, class := evaluateCandidate(target, view, snapshot, requirements)
		eligible := len(reasons) == 0

		evaluated = append(evaluated, CandidateEvaluation{
			Target:         target,
			Eligible:       eligible,
			Reasons:        reasons,
			ExclusionClass: class,
		})
	}

	result.Evaluated = evaluated
	result.Eligible = collectEligible(evaluated)
	result.Coverage = buildCoverage(snapshot)

	return result
}

func evaluateCandidate(target Target, view catalog.View, snapshot Snapshot, requirements RequirementStatus) ([]ReasonCode, ExclusionClass) {
	reasons := make([]ReasonCode, 0, 4)
	class := ExclusionNotApplicable

	if providerReasons, providerClass := evaluateProvider(target.ProviderID, snapshot); len(providerReasons) > 0 {
		return append(reasons, providerReasons...), providerClass
	}

	if view.Presence == catalog.PresenceMissing {
		reasons = append(reasons, ReasonModelMissing)
		return reasons, ExclusionIncompatible
	}

	if view.Freshness != catalog.FreshnessFresh {
		reasons = append(reasons, ReasonCatalogStale)
		return reasons, ExclusionInsufficient
	}

	for _, capability := range requirements.RequiredCapabilities {
		claim, exists := view.Entry.Capabilities[capability]
		if !exists || claim.State == catalog.CapabilityUnknown {
			reasons = append(reasons, ReasonCapabilityUnknown)
			class = ExclusionInsufficient
			continue
		}

		if claim.State == catalog.CapabilityUnsupported {
			reasons = append(reasons, ReasonCapabilityUnsupported)
			class = ExclusionIncompatible
		}
	}

	if requirements.MaxOutputTokens != nil {
		if view.Entry.Limits.Output == nil {
			reasons = append(reasons, ReasonLimitUnknown)
			if class != ExclusionIncompatible {
				class = ExclusionInsufficient
			}
		} else if *requirements.MaxOutputTokens > *view.Entry.Limits.Output {
			reasons = append(reasons, ReasonLimitExceeded)
			class = ExclusionIncompatible
		}
	}

	if len(reasons) > 0 {
		if class == ExclusionNotApplicable {
			class = ExclusionInsufficient
		}

		return dedupeReasons(reasons), class
	}

	return nil, ExclusionNotApplicable
}

func evaluateProvider(providerID string, snapshot Snapshot) ([]ReasonCode, ExclusionClass) {
	admin, adminKnown := snapshot.Admin[providerID]
	runtimeFacts, runtimeKnown := snapshot.Runtime[providerID]

	if !adminKnown && !runtimeKnown {
		return []ReasonCode{ReasonProviderUnknown}, ExclusionIncompatible
	}

	if runtimeKnown && runtimeFacts.Bound && (!adminKnown || !admin.Known) {
		return []ReasonCode{ReasonCredentialMissing}, ExclusionInsufficient
	}

	if adminKnown && !admin.Enabled {
		return []ReasonCode{ReasonProviderDisabled}, ExclusionIncompatible
	}

	if adminKnown && !admin.HasCredential {
		return []ReasonCode{ReasonCredentialMissing}, ExclusionIncompatible
	}

	if !runtimeKnown || !runtimeFacts.Bound {
		return []ReasonCode{ReasonProviderUnbound}, ExclusionIncompatible
	}

	if !runtimeFacts.ResponsesSupported {
		return []ReasonCode{ReasonResponsesUnsupported}, ExclusionIncompatible
	}

	if !runtimeFacts.RuntimeEnabled {
		return []ReasonCode{ReasonProviderUnavailable}, ExclusionIncompatible
	}

	return nil, ExclusionNotApplicable
}

func buildCoverage(snapshot Snapshot) CoverageEvidence {
	providerIDs := make(map[string]struct{})

	for id := range snapshot.Admin {
		if providers.ExcludedFromCatalog(id) {
			continue
		}
		providerIDs[id] = struct{}{}
	}

	for id := range snapshot.Runtime {
		if providers.ExcludedFromCatalog(id) {
			continue
		}
		providerIDs[id] = struct{}{}
	}

	for id := range snapshot.Sync {
		if providers.ExcludedFromCatalog(id) {
			continue
		}
		providerIDs[id] = struct{}{}
	}

	ids := make([]string, 0, len(providerIDs))
	for id := range providerIDs {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	coverage := make([]ProviderCoverage, 0, len(ids))
	complete := true

	for _, id := range ids {
		reasons, _ := evaluateProvider(id, snapshot)
		passes := len(reasons) == 0

		sync, ok := snapshot.Sync[id]
		fresh := ok && providerCatalogFresh(sync, snapshot.CapturedAt, snapshot.StaleAfter)

		entry := ProviderCoverage{
			ProviderID:           id,
			PassesProviderChecks: passes,
			CatalogFreshAccepted: fresh,
		}

		if ok {
			entry.SyncOutcome = sync.Outcome
		}

		if passes && !fresh {
			complete = false
		}

		coverage = append(coverage, entry)
	}

	return CoverageEvidence{
		Providers:                        coverage,
		CompleteFreshCoverageForEligible: complete,
	}
}

func providerCatalogFresh(sync catalog.ProviderSyncState, now time.Time, staleAfter time.Duration) bool {
	if sync.LastSuccessAt == nil {
		return false
	}

	if staleAfter <= 0 {
		staleAfter = 24 * time.Hour
	}

	return now.Sub(*sync.LastSuccessAt) <= staleAfter
}

func collectEligible(evaluated []CandidateEvaluation) []Target {
	eligible := make([]Target, 0, len(evaluated))

	for _, candidate := range evaluated {
		if candidate.Eligible {
			eligible = append(eligible, candidate.Target)
		}
	}

	return eligible
}

func dedupeReasons(reasons []ReasonCode) []ReasonCode {
	if len(reasons) <= 1 {
		return reasons
	}

	sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })

	unique := reasons[:1]
	for _, reason := range reasons[1:] {
		if reason != unique[len(unique)-1] {
			unique = append(unique, reason)
		}
	}

	return unique
}
