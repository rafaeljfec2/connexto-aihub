package routing

import (
	"net/http"
	"strings"
)

// AutoShadowMode compares old vs new selector without changing execution.
var AutoShadowMode bool

// AutoGate0Enforced blocks aihub/auto when Gate 0 is unsatisfied (opt-in via AIHUB_AUTO_GATE0_ENFORCED).
var AutoGate0Enforced = false

// selectVirtualAuto resolves aihub/auto using allowlist, bands, and AW-RR.
func selectVirtualAuto(decision RoutingDecision, ctx RoutingContext, snapshot Snapshot, eligibility EligibilityResult) RoutingDecision {
	understanding := ctx.Understanding
	profile := understanding.OptimizationProfile
	if profile == "" {
		profile = ProfileBalanced
	}

	if AutoGate0Enforced && !AutoGate0Satisfied() {
		return reject(decision, RejectRoutingUnavailable, []string{"gate0_unsatisfied", "profile:" + string(profile)})
	}

	allowlisted := FilterEligibleToAutoAllowlistForSnapshot(eligibility.Eligible, snapshot)
	if len(allowlisted) == 0 {
		return reject(decision, classifyZeroEligible(eligibility.Evaluated), []string{"allowlist_miss", "profile:" + string(profile)})
	}

	quota := snapshot.AutoQuota
	if AutoShadowMode {
		legacy := selectVirtualAutoLegacy(decision, snapshot, eligibility)
		decision.ShadowLegacyModel = legacySelectedModel(legacy)
	}

	bandB, bandReasons := FilterAutoBandBCandidates(understanding, allowlisted)
	if len(bandB) == 0 {
		criteria := append(bandReasons, "contract_band", "profile:"+string(profile))
		return reject(decision, RejectNoEligibleModel, criteria)
	}

	bandB, ctxReasons := FilterAutoContextWindow(bandB, understanding, snapshot)
	bandReasons = append(bandReasons, ctxReasons...)
	if len(bandB) == 0 {
		criteria := append(bandReasons, "context_window", "profile:"+string(profile))
		return reject(decision, RejectNoEligibleModel, criteria)
	}

	poolID, poolReasons := SelectPoolAWRR(bandB, quota)
	if poolID == "" {
		return reject(decision, RejectNoEligibleModel, append(poolReasons, "profile:"+string(profile)))
	}

	selected, modelReasons := SelectModelInPool(poolID, bandB, snapshot, understanding)
	if selected.ModelID == "" {
		return reject(decision, RejectNoEligibleModel, append(modelReasons, "profile:"+string(profile)))
	}

	ProvisionalDebit(poolID)

	decision.Selected = &selected
	decision.Origin = OriginVirtualAuto
	decision.AppliedProfile = profile
	decision.RoutingReasonCode = "awrr_pool_select"
	decision.SelectedPool = string(poolID)
	if st, ok := quota.Pools[poolID]; ok && st.Pressure != nil {
		p := *st.Pressure
		decision.PoolPressure = &p
	}
	if st, ok := quota.Pools[poolID]; ok && st.Weight != nil {
		w := *st.Weight
		decision.PoolWeight = &w
	}
	if st, ok := quota.Pools[poolID]; ok && st.ReserveBreached {
		modelReasons = append(modelReasons, "quota_reserve_breached")
	}

	criteria := []string{
		"rule:auto_awrr",
		"pool:" + string(poolID),
		"profile:" + string(profile),
		"reason:" + decision.RoutingReasonCode,
	}
	criteria = append(criteria, poolReasons...)
	criteria = append(criteria, modelReasons...)
	decision.Criteria = criteria

	if AutoShadowMode {
		decision.ShadowCandidateModel = selected.ModelID
		decision.ShadowCandidatePool = string(poolID)
	}
	return decision
}

func legacySelectedModel(decision RoutingDecision) string {
	if decision.Selected == nil {
		return ""
	}
	return decision.Selected.ModelID
}

// selectVirtualAutoLegacy preserves the pre-MVP administrative priority selector.
func selectVirtualAutoLegacy(decision RoutingDecision, snapshot Snapshot, eligibility EligibilityResult) RoutingDecision {
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
		for _, priorityTarget := range DefaultVirtualPriority {
			if containsTarget(eligibility.Eligible, priorityTarget) {
				selected := priorityTarget
				decision.Selected = &selected
				decision.Origin = OriginVirtualAuto
				decision.Criteria = []string{"rule:virtual_auto_ranked_priority", "profile:balanced"}
				return decision
			}
		}
		target := eligibility.Eligible[0]
		decision.Selected = &target
		decision.Origin = OriginVirtualAuto
		decision.Criteria = []string{"rule:virtual_auto_snapshot_fallback", "profile:balanced"}
		return decision
	}
}

// MapAutoGate0HTTPStatus maps gate rejection to 503 when enforced.
func MapAutoGate0HTTPStatus(criteria []string) (int, bool) {
	if !AutoGate0Enforced || AutoGate0Satisfied() {
		return 0, false
	}
	for _, c := range criteria {
		if strings.Contains(c, "gate0_unsatisfied") {
			return http.StatusServiceUnavailable, true
		}
	}
	return 0, false
}
