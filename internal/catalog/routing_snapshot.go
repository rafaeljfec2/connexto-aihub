package catalog

import (
	"sort"

	"aihub/internal/providers"
)

// RoutingSnapshotBundle is a point-in-time catalog read for routing eligibility.
// Views and Sync are captured under one read lock.
type RoutingSnapshotBundle struct {
	Views []View
	Sync  map[string]ProviderSyncState
}

// CaptureRoutingSnapshot copies catalog entries and provider sync metadata under
// one read lock, then derives model views from the supplied providerStates.
// providerStates must be captured before this call and remain unchanged for the
// duration of one routing evaluation.
func (m *Manager) CaptureRoutingSnapshot(providerStates map[string]providers.State) RoutingSnapshotBundle {
	now := m.now()

	m.mu.RLock()
	entries := make([]ModelEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		entries = append(entries, cloneEntry(entry))
	}

	syncStates := make(map[string]ProviderSyncState, len(m.syncStates))
	for providerID, state := range m.syncStates {
		syncStates[providerID] = state
	}
	m.mu.RUnlock()

	views := make([]View, 0, len(entries))

	for _, entry := range entries {
		state, ok := syncStates[entry.ProviderID]
		if !ok {
			state = ProviderSyncState{Outcome: SyncOutcomeNever, ErrorClass: SyncErrorNone}
		}

		providerState, ok := providerStates[entry.ProviderID]
		if !ok {
			providerState = providers.StateUnknown
		}

		views = append(views, DeriveView(entry, state, providerState, now, m.staleAfter))
	}

	sort.Slice(views, func(left, right int) bool {
		if views[left].Entry.ProviderID != views[right].Entry.ProviderID {
			return views[left].Entry.ProviderID < views[right].Entry.ProviderID
		}

		return views[left].Entry.ModelID < views[right].Entry.ModelID
	})

	return RoutingSnapshotBundle{Views: views, Sync: syncStates}
}
