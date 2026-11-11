package routing

import (
	"math"
	"sort"
	"sync"
)

type awrrState struct {
	mu          sync.Mutex
	poolCurrent map[AutoPoolID]float64
	poolIndex   int
}

var autoBalancer = &awrrState{poolCurrent: make(map[AutoPoolID]float64)}

// SelectPoolAWRR picks one pool using smooth weighted round-robin with dynamic weights.
func SelectPoolAWRR(candidates []Target, quota AutoQuotaView) (AutoPoolID, []string) {
	pools := uniquePools(candidates, quota)
	if len(pools) == 0 {
		return "", []string{"quota_unknown_all"}
	}

	known, hasWeights := filterKnownPools(pools, quota)
	var reasons []string
	if len(known) == 0 {
		reasons = append(reasons, "quota_unknown_all")
		return rotateUnknownPools(pools), reasons
	}
	if !hasWeights {
		reasons = append(reasons, "quota_unknown_all")
		return rotateUnknownPools(known), reasons
	}

	if len(known) == 1 {
		return known[0], reasons
	}

	applyReservePolicy(&known, quota)

	weights := poolWeights(known, quota)
	id, tie := autoBalancer.selectSWRR(known, weights)
	if tie {
		reasons = append(reasons, "tie_order")
	}
	return id, reasons
}

func uniquePools(candidates []Target, quota AutoQuotaView) []AutoPoolID {
	seen := make(map[AutoPoolID]struct{})
	out := make([]AutoPoolID, 0, 4)
	for _, t := range candidates {
		id := PoolIDForTarget(t)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func filterKnownPools(pools []AutoPoolID, quota AutoQuotaView) ([]AutoPoolID, bool) {
	out := make([]AutoPoolID, 0, len(pools))
	unknown := make([]AutoPoolID, 0, len(pools))
	hasWeights := false
	for _, id := range pools {
		st, ok := quota.Pools[id]
		if ok && st.Weight != nil {
			out = append(out, id)
			hasWeights = true
			continue
		}
		unknown = append(unknown, id)
	}
	if len(out) == 0 {
		return unknown, false
	}
	return out, hasWeights
}

func applyReservePolicy(pools *[]AutoPoolID, quota AutoQuotaView) {
	if len(*pools) <= 1 {
		return
	}
	healthy := make([]AutoPoolID, 0, len(*pools))
	for _, id := range *pools {
		st := quota.Pools[id]
		if st.Pressure != nil && *st.Pressure < 0.9 {
			healthy = append(healthy, id)
		}
	}
	if len(healthy) == 0 {
		return
	}
	filtered := make([]AutoPoolID, 0, len(*pools))
	for _, id := range *pools {
		st := quota.Pools[id]
		if st.Pressure != nil && *st.Pressure >= 0.9 {
			continue
		}
		filtered = append(filtered, id)
	}
	if len(filtered) > 0 {
		*pools = filtered
	}
}

func poolWeights(pools []AutoPoolID, quota AutoQuotaView) map[AutoPoolID]float64 {
	weights := make(map[AutoPoolID]float64, len(pools))
	for _, id := range pools {
		st := quota.Pools[id]
		penalty := float64(autoRuntimeState.provisionalCount(id)) * 0.02
		if w, ok := EffectivePoolWeight(st, penalty); ok {
			weights[id] = w
		}
	}
	return weights
}

func rotateUnknownPools(pools []AutoPoolID) AutoPoolID {
	autoBalancer.mu.Lock()
	defer autoBalancer.mu.Unlock()
	if len(pools) == 0 {
		return ""
	}
	idx := autoBalancer.poolIndex % len(pools)
	autoBalancer.poolIndex++
	return pools[idx]
}

func (b *awrrState) selectSWRR(pools []AutoPoolID, weights map[AutoPoolID]float64) (AutoPoolID, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var selected AutoPoolID
	maxCurrent := -1.0
	tie := false

	for _, id := range pools {
		w := weights[id]
		if w <= 0 {
			w = 0.05
		}
		b.poolCurrent[id] += w
		if b.poolCurrent[id] > maxCurrent {
			maxCurrent = b.poolCurrent[id]
			selected = id
			tie = false
		} else if math.Abs(b.poolCurrent[id]-maxCurrent) < 1e-9 && id != selected {
			tie = true
		}
	}
	if selected != "" {
		total := 0.0
		for _, id := range pools {
			total += weights[id]
		}
		if total <= 0 {
			total = float64(len(pools)) * 0.05
		}
		b.poolCurrent[selected] -= total
	}
	return selected, tie
}

// SelectModelInPool chooses a model inside the pool using bands, profile, and tie order.
func SelectModelInPool(pool AutoPoolID, candidates []Target, understanding RequestUnderstanding) (Target, []string) {
	inPool := make([]Target, 0, len(candidates))
	for _, t := range candidates {
		if PoolIDForTarget(t) == pool {
			inPool = append(inPool, t)
		}
	}
	if len(inPool) == 0 {
		return Target{}, []string{"allowlist_miss"}
	}

	bandA := make([]Target, 0, len(inPool))
	reasons := make([]string, 0, 4)
	for _, t := range inPool {
		ok, code := PassesAutoContractBand(understanding, t)
		if ok {
			bandA = append(bandA, t)
		} else if code != "" {
			reasons = append(reasons, code)
		}
	}
	if len(bandA) == 0 {
		return Target{}, reasons
	}

	ordered := orderByAdminTie(bandA)
	if understanding.OptimizationProfile == ProfileEfficiency {
		ordered = deprioritizeGeminiShortCalls(ordered, understanding)
	}
	if understanding.OptimizationProfile == ProfileIntelligence {
		reasons = append(reasons, "intelligence_profile_no_evidence")
	}

	return ordered[0], reasons
}

func orderByAdminTie(candidates []Target) []Target {
	set := make(map[string]Target, len(candidates))
	for _, t := range candidates {
		set[targetKey(t)] = t
	}
	out := make([]Target, 0, len(candidates))
	for _, pref := range AutoAdminTieOrder {
		if t, ok := set[targetKey(pref)]; ok {
			out = append(out, t)
			delete(set, targetKey(pref))
		}
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, set[k])
	}
	return out
}

func deprioritizeGeminiShortCalls(ordered []Target, u RequestUnderstanding) []Target {
	if u.EstimatedInputTokens >= 30000 || u.IsStreaming {
		return ordered
	}
	gemini := Target{ProviderID: "antigravity", ModelID: "antigravity:gemini-3.8-flash"}
	var tail []Target
	var front []Target
	for _, t := range ordered {
		if t == gemini {
			tail = append(tail, t)
			continue
		}
		front = append(front, t)
	}
	return append(front, tail...)
}
