package routing

import (
	"time"

	"aihub/internal/providers"
)

// AutoPoolID identifies a real quota consumption pool.
type AutoPoolID string

const (
	PoolOpenCodeShared  AutoPoolID = "pool:opencode-go:subscription"
	PoolOpenRouterKey   AutoPoolID = "pool:openrouter:inference_key"
	PoolCursorBrowser   AutoPoolID = "pool:cursor:browser_account"
	poolAntigravityPref              = "pool:antigravity:model:"
)

// QuotaPressureState describes normalized pool pressure for AW-RR.
type QuotaPressureState struct {
	PoolID          AutoPoolID
	Pressure        *float64 // 0..1 known; nil unknown
	Weight          *float64
	Stale           bool
	Exhausted       bool
	Age             time.Duration
	ReserveBreached bool
	ReasonCodes     []string
}

// AutoQuotaView maps pools and per-target pressure from cached quota snapshots.
type AutoQuotaView struct {
	Pools    map[AutoPoolID]QuotaPressureState
	ByTarget map[string]QuotaPressureState
}

// PoolIDForTarget resolves the consumption pool for an allowlist target.
func PoolIDForTarget(t Target) AutoPoolID {
	switch t.ProviderID {
	case providers.OpenCodeGoID:
		return PoolOpenCodeShared
	case providers.OpenRouterID:
		return PoolOpenRouterKey
	case providers.CursorID:
		return PoolCursorBrowser
	case providers.AntigravityID:
		return AutoPoolID(poolAntigravityPref + t.ModelID)
	default:
		return ""
	}
}

// BuildAutoQuotaView normalizes cached provider quota snapshots (no HTTP).
func BuildAutoQuotaView(snapshots []providers.ProviderQuotaSnapshot, now time.Time, staleTTL time.Duration) AutoQuotaView {
	view := AutoQuotaView{
		Pools:    make(map[AutoPoolID]QuotaPressureState),
		ByTarget: make(map[string]QuotaPressureState),
	}
	if staleTTL <= 0 {
		staleTTL = 2 * time.Minute
	}

	var openCode *providers.ProviderQuotaSnapshot
	var openRouter *providers.ProviderQuotaSnapshot
	var cursor *providers.ProviderQuotaSnapshot
	var antigravity *providers.ProviderQuotaSnapshot

	for i := range snapshots {
		snap := snapshots[i]
		switch snap.ProviderID {
		case providers.OpenCodeGoID:
			if openCode == nil {
				openCode = &snap
			}
		case providers.OpenRouterID:
			if openRouter == nil {
				openRouter = &snap
			}
		case providers.CursorID:
			if cursor == nil {
				cursor = &snap
			}
		case providers.AntigravityID:
			if antigravity == nil {
				antigravity = &snap
			}
		}
	}

	if openCode != nil {
		state := pressureOpenCodePool(openCode, now, staleTTL)
		view.Pools[PoolOpenCodeShared] = state
		for _, t := range AutoMVPAllowlist {
			if t.ProviderID == providers.OpenCodeGoID {
				view.ByTarget[targetKey(t)] = state
			}
		}
	}
	if openRouter != nil {
		state := pressureOpenRouterPool(openRouter, now, staleTTL)
		view.Pools[PoolOpenRouterKey] = state
		for _, t := range AutoMVPAllowlist {
			if t.ProviderID == providers.OpenRouterID {
				view.ByTarget[targetKey(t)] = state
			}
		}
	}
	if cursor != nil {
		state := pressureCursorPool(cursor, now, staleTTL)
		view.Pools[PoolCursorBrowser] = state
		for _, t := range AutoMVPAllowlist {
			if t.ProviderID == providers.CursorID {
				view.ByTarget[targetKey(t)] = state
			}
		}
	}
	if antigravity != nil {
		for _, t := range AutoMVPAllowlist {
			if t.ProviderID != providers.AntigravityID {
				continue
			}
			pool := PoolIDForTarget(t)
			state := pressureAntigravityModel(antigravity, t.ModelID, pool, now, staleTTL)
			view.Pools[pool] = state
			view.ByTarget[targetKey(t)] = state
		}
	}

	return view
}
