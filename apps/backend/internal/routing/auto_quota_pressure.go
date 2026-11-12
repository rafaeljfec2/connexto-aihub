package routing

import (
	"math"
	"strings"
	"time"

	"aihub/internal/providers"
)

func snapshotAge(snap *providers.ProviderQuotaSnapshot, now time.Time) (time.Duration, bool) {
	if snap == nil {
		return 0, false
	}
	if snap.Stale {
		return 0, true
	}
	fetchedAt := strings.TrimSpace(snap.Source.FetchedAt)
	if fetchedAt == "" {
		return 0, true
	}
	parsed, err := time.Parse(time.RFC3339, fetchedAt)
	if err != nil {
		return 0, true
	}
	return now.Sub(parsed.UTC()), false
}

func clampWeight(p float64) float64 {
	w := 1 - p
	if w < 0.05 {
		return 0.05
	}
	if w > 1 {
		return 1
	}
	return w
}

func pressureOpenCodePool(snap *providers.ProviderQuotaSnapshot, now time.Time, staleTTL time.Duration) QuotaPressureState {
	state := QuotaPressureState{PoolID: PoolOpenCodeShared, ReasonCodes: []string{"pool_shared_opencode"}}
	age, _ := snapshotAge(snap, now)
	state.Age = age
	state.Stale = snap.Stale || age > staleTTL
	maxPct := 0.0
	found := false
	for _, m := range snap.Quotas {
		if m.ConsumedPercent == nil {
			continue
		}
		switch m.Metric {
		case "go_rolling", "go_weekly", "go_monthly":
			found = true
			if *m.ConsumedPercent > maxPct {
				maxPct = *m.ConsumedPercent
			}
		}
	}
	if !found || snap.Status != providers.QuotaStatusAvailable {
		state.ReasonCodes = append(state.ReasonCodes, "quota_unknown")
		return state
	}
	p := maxPct / 100
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	state.Pressure = &p
	w := clampWeight(p)
	state.Weight = &w
	state.Exhausted = p >= 0.99
	return state
}

func pressureOpenRouterPool(snap *providers.ProviderQuotaSnapshot, now time.Time, staleTTL time.Duration) QuotaPressureState {
	state := QuotaPressureState{PoolID: PoolOpenRouterKey}
	age, _ := snapshotAge(snap, now)
	state.Age = age
	state.Stale = snap.Stale || age > staleTTL
	var limit, remaining *float64
	for _, m := range snap.Quotas {
		switch m.Metric {
		case "limit":
			limit = m.Limit
		case "limit_remaining":
			remaining = m.Remaining
		}
	}
	if limit == nil || remaining == nil || *limit <= 0 {
		state.ReasonCodes = append(state.ReasonCodes, "quota_unknown")
		return state
	}
	p := 1 - (*remaining / *limit)
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	state.Pressure = &p
	w := clampWeight(p)
	state.Weight = &w
	state.Exhausted = *remaining <= 0
	return state
}

func pressureCursorPool(snap *providers.ProviderQuotaSnapshot, now time.Time, staleTTL time.Duration) QuotaPressureState {
	state := QuotaPressureState{PoolID: PoolCursorBrowser}
	age, _ := snapshotAge(snap, now)
	state.Age = age
	state.Stale = snap.Stale || age > staleTTL
	var weeklyPct *float64
	for _, m := range snap.Quotas {
		if m.Metric == "weekly_used_percent" && m.ConsumedPercent != nil {
			weeklyPct = m.ConsumedPercent
			break
		}
	}
	if weeklyPct == nil || snap.Status != providers.QuotaStatusAvailable {
		state.ReasonCodes = append(state.ReasonCodes, "quota_unknown")
		return state
	}
	p := *weeklyPct / 100
	state.Pressure = &p
	w := clampWeight(p)
	state.Weight = &w
	return state
}

func pressureAntigravityModel(snap *providers.ProviderQuotaSnapshot, modelID string, pool AutoPoolID, now time.Time, staleTTL time.Duration) QuotaPressureState {
	state := QuotaPressureState{PoolID: pool}
	age, _ := snapshotAge(snap, now)
	state.Age = age
	state.Stale = snap.Stale || age > staleTTL
	metricID := strings.TrimPrefix(modelID, "antigravity:")
	var fraction *float64
	for _, m := range snap.Quotas {
		if m.RemainingFraction != nil && (m.Metric == metricID || strings.Contains(strings.ToLower(m.Label), metricID)) {
			fraction = m.RemainingFraction
			break
		}
	}
	if fraction == nil {
		for _, m := range snap.Quotas {
			if m.RemainingFraction != nil && m.Metric == modelID {
				fraction = m.RemainingFraction
				break
			}
		}
	}
	if fraction == nil {
		state.ReasonCodes = append(state.ReasonCodes, "quota_unknown")
		return state
	}
	p := 1 - *fraction
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	state.Pressure = &p
	w := clampWeight(p)
	state.Weight = &w
	state.Exhausted = *fraction <= 0.01
	if *fraction < 0.10 {
		state.ReserveBreached = true
	}
	return state
}

func pressureAntigravityAccountPool(snap *providers.ProviderQuotaSnapshot, now time.Time, staleTTL time.Duration) QuotaPressureState {
	accountID := snap.AccountID
	if accountID == "" {
		accountID = snap.ProviderID
	}
	pool := AccountPoolID(accountID)
	state := QuotaPressureState{PoolID: pool}
	age, _ := snapshotAge(snap, now)
	state.Age = age
	state.Stale = snap.Stale || age > staleTTL

	worstFraction := 2.0
	for _, m := range snap.Quotas {
		if m.RemainingFraction != nil && *m.RemainingFraction < worstFraction {
			worstFraction = *m.RemainingFraction
		}
	}
	if worstFraction > 1.0 {
		state.ReasonCodes = append(state.ReasonCodes, "quota_unknown")
		return state
	}
	p := 1 - worstFraction
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	state.Pressure = &p
	w := clampWeight(p)
	state.Weight = &w
	state.Exhausted = worstFraction <= 0.01
	if worstFraction < 0.10 {
		state.ReserveBreached = true
	}
	return state
}

// EffectivePoolWeight applies reserve policy and provisional debits (advisory).
func EffectivePoolWeight(state QuotaPressureState, provisionalPenalty float64) (float64, bool) {
	if state.Weight == nil {
		return 0, false
	}
	w := *state.Weight
	if provisionalPenalty > 0 {
		w = math.Max(0.05, w-provisionalPenalty)
	}
	return w, true
}
