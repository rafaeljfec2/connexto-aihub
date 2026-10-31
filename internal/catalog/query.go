package catalog

import (
	"sort"

	"aihub/internal/providers"
)

// Filter narrows a catalog listing. Zero values mean "no filter".
type Filter struct {
	ProviderID   string
	Presence     *Presence
	ListingState *ListingState
	Capability   string
}

// List returns derived views sorted by provider and model id. It copies the
// accepted state under a read lock and performs no I/O.
func (m *Manager) List(filter Filter) []View {
	now := m.now()
	statuses := providerStates(m.statuses())

	m.mu.RLock()
	entries := make([]ModelEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}

	states := make(map[string]ProviderSyncState, len(m.syncStates))
	for providerID, state := range m.syncStates {
		states[providerID] = state
	}
	m.mu.RUnlock()

	views := make([]View, 0, len(entries))

	for _, entry := range entries {
		if filter.ProviderID != "" && entry.ProviderID != filter.ProviderID {
			continue
		}

		state, ok := states[entry.ProviderID]
		if !ok {
			state = ProviderSyncState{Outcome: SyncOutcomeNever, ErrorClass: SyncErrorNone}
		}

		providerState, ok := statuses[entry.ProviderID]
		if !ok {
			providerState = providers.StateUnknown
		}

		view := DeriveView(cloneEntry(entry), state, providerState, now, m.staleAfter)

		if filter.Presence != nil && view.Presence != *filter.Presence {
			continue
		}

		if filter.ListingState != nil && view.ListingState != *filter.ListingState {
			continue
		}

		if filter.Capability != "" {
			claim, exists := view.Entry.Capabilities[filter.Capability]
			if !exists || claim.State != CapabilitySupported {
				continue
			}
		}

		views = append(views, view)
	}

	sort.Slice(views, func(left, right int) bool {
		if views[left].Entry.ProviderID != views[right].Entry.ProviderID {
			return views[left].Entry.ProviderID < views[right].Entry.ProviderID
		}

		return views[left].Entry.ModelID < views[right].Entry.ModelID
	})

	return views
}

// Get returns one derived view.
func (m *Manager) Get(providerID, modelID string) (View, bool) {
	now := m.now()
	statuses := providerStates(m.statuses())

	m.mu.RLock()
	entry, ok := m.entries[ModelKey{ProviderID: providerID, ModelID: modelID}]
	state := m.syncStates[providerID]
	m.mu.RUnlock()

	if !ok {
		return View{}, false
	}

	providerState, exists := statuses[providerID]
	if !exists {
		providerState = providers.StateUnknown
	}

	return DeriveView(cloneEntry(entry), state, providerState, now, m.staleAfter), true
}

// Providers returns provider-level views sorted by provider id.
func (m *Manager) Providers() []ProviderView {
	now := m.now()
	statuses := providerStates(m.statuses())

	m.mu.RLock()
	states := make(map[string]ProviderSyncState, len(m.syncStates))
	for providerID, state := range m.syncStates {
		states[providerID] = state
	}

	counts := make(map[string][3]int, len(m.syncStates))
	for _, entry := range m.entries {
		current := counts[entry.ProviderID]
		current[0]++

		if entry.MissingSince == nil {
			current[1]++
		} else {
			current[2]++
		}

		counts[entry.ProviderID] = current
	}
	m.mu.RUnlock()

	views := make([]ProviderView, 0, len(states))

	for providerID, state := range states {
		providerState, ok := statuses[providerID]
		if !ok {
			providerState = providers.StateUnknown
		}

		current := counts[providerID]

		views = append(views, ProviderView{
			ProviderID:    providerID,
			ProviderState: providerState,
			Freshness:     deriveFreshness(state, now, m.staleAfter),
			Sync:          state,
			ModelsKnown:   current[0],
			ModelsPresent: current[1],
			ModelsMissing: current[2],
		})
	}

	sort.Slice(views, func(left, right int) bool {
		return views[left].ProviderID < views[right].ProviderID
	})

	return views
}

// cloneEntry isolates callers from the manager's internal maps, slices and
// pointers. It is deliberately shallow for immutable values and deep for the
// mutable containers defined by ModelEntry.
func cloneEntry(entry ModelEntry) ModelEntry {
	capabilities := make(map[string]CapabilityClaim, len(entry.Capabilities))
	for name, claim := range entry.Capabilities {
		capabilities[name] = claim
	}

	entry.Capabilities = capabilities
	entry.Limits = Limits{
		Context: cloneInt64(entry.Limits.Context),
		Input:   cloneInt64(entry.Limits.Input),
		Output:  cloneInt64(entry.Limits.Output),
	}
	entry.ProviderMetadata = cloneRaw(entry.ProviderMetadata)

	if entry.MissingSince != nil {
		missingSince := *entry.MissingSince
		entry.MissingSince = &missingSince
	}

	if entry.Modalities != nil {
		entry.Modalities = &Modalities{
			Input:  cloneStrings(entry.Modalities.Input),
			Output: cloneStrings(entry.Modalities.Output),
		}
	}

	return entry
}

func providerStates(statuses []providers.Status) map[string]providers.State {
	states := make(map[string]providers.State, len(statuses))

	for _, status := range statuses {
		states[status.ID] = status.State
	}

	return states
}
