package catalog

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"time"

	"aihub/internal/providers"
)

func (m *Manager) successCandidate(providerID string, descriptors []providers.ModelDescriptor, now time.Time) (map[ModelKey]ModelEntry, ProviderSyncState, SyncCounts) {
	m.mu.RLock()
	previous := m.entries
	previousSync := m.syncStates[providerID]
	m.mu.RUnlock()

	return buildCandidate(providerID, previous, previousSync, descriptors, now)
}

func buildCandidate(providerID string, previous map[ModelKey]ModelEntry, previousSync ProviderSyncState, descriptors []providers.ModelDescriptor, now time.Time) (map[ModelKey]ModelEntry, ProviderSyncState, SyncCounts) {
	next := copyEntries(previous)
	seen := make(map[ModelKey]struct{}, len(descriptors))
	counts := SyncCounts{Discovered: len(descriptors)}

	for _, descriptor := range descriptors {
		key := ModelKey{ProviderID: providerID, ModelID: descriptor.ID}
		entry := NormalizeEntry(providerID, descriptor, now)

		prior, known := previous[key]
		if known {
			entry.FirstSeenAt = prior.FirstSeenAt

			switch {
			case prior.MissingSince != nil:
				counts.Updated++
			case sameContent(prior, entry):
				counts.Unchanged++
			default:
				counts.Updated++
			}
		} else {
			counts.Added++
		}

		entry.LastSeenAt = now
		entry.MissingSince = nil
		next[key] = entry
		seen[key] = struct{}{}
	}

	for key, entry := range previous {
		if key.ProviderID != providerID {
			continue
		}

		if entry.MissingSince != nil {
			continue
		}

		if _, present := seen[key]; present {
			continue
		}

		counts.Missing++
		missingSince := now
		entry.MissingSince = &missingSince
		next[key] = entry
	}

	state := previousSync
	state.LastAttemptAt = &now
	state.LastSuccessAt = &now
	state.Outcome = SyncOutcomeSuccess
	state.ErrorClass = SyncErrorNone
	state.UpstreamStatus = 200
	state.Counts = counts

	return next, state, counts
}

func (m *Manager) failureCandidate(providerID string, outcome SyncOutcome, class SyncErrorClass, upstreamStatus int, now time.Time) (map[ModelKey]ModelEntry, ProviderSyncState, SyncCounts) {
	m.mu.RLock()
	entries := copyEntries(m.entries)
	state := m.syncStates[providerID]
	m.mu.RUnlock()

	state.LastAttemptAt = &now
	state.Outcome = outcome
	state.ErrorClass = class
	state.UpstreamStatus = upstreamStatus

	return entries, state, state.Counts
}

func (m *Manager) rejectedEmptyCandidate(providerID string, now time.Time) (map[ModelKey]ModelEntry, ProviderSyncState, SyncCounts) {
	m.mu.RLock()
	entries := copyEntries(m.entries)
	state := m.syncStates[providerID]
	m.mu.RUnlock()

	state.LastAttemptAt = &now
	state.Outcome = SyncOutcomeRejectedEmpty
	state.ErrorClass = SyncErrorEmptyCatalog
	state.UpstreamStatus = 200

	return entries, state, state.Counts
}

func classifySyncError(err error) (SyncOutcome, SyncErrorClass, int) {
	if errors.Is(err, context.DeadlineExceeded) {
		return SyncOutcomeFailed, SyncErrorTimeout, 0
	}

	var httpErr *providers.UpstreamHTTPError
	if errors.As(err, &httpErr) {
		return SyncOutcomeFailed, SyncErrorHTTPStatus, httpErr.StatusCode
	}

	var decodeErr *providers.DecodeError
	if errors.As(err, &decodeErr) {
		return SyncOutcomeFailed, SyncErrorInvalidPayload, decodeErr.StatusCode
	}

	return SyncOutcomeFailed, SyncErrorTransport, 0
}

func sameContent(first, second ModelEntry) bool {
	return first.ProviderID == second.ProviderID &&
		first.ModelID == second.ModelID &&
		first.CanonicalID == second.CanonicalID &&
		first.Name == second.Name &&
		first.Type == second.Type &&
		first.Family == second.Family &&
		reflect.DeepEqual(first.Modalities, second.Modalities) &&
		reflect.DeepEqual(first.Limits, second.Limits) &&
		reflect.DeepEqual(first.Capabilities, second.Capabilities)
	// ProviderMetadata is preserved but is not part of the normalized content:
	// a provider reordering JSON keys must not mark every model as updated.
}

func buildSnapshot(entries map[ModelKey]ModelEntry, states map[string]ProviderSyncState, now time.Time) Snapshot {
	providers := make(map[string]ProviderSnapshot, len(states))

	for providerID, state := range states {
		providers[providerID] = ProviderSnapshot{Sync: state}
	}

	for key, entry := range entries {
		provider, ok := providers[key.ProviderID]
		if !ok {
			provider = ProviderSnapshot{
				Sync: ProviderSyncState{Outcome: SyncOutcomeNever, ErrorClass: SyncErrorNone},
			}
		}

		provider.Models = append(provider.Models, entry)
		providers[key.ProviderID] = provider
	}

	for providerID, provider := range providers {
		sort.Slice(provider.Models, func(left, right int) bool {
			return provider.Models[left].ModelID < provider.Models[right].ModelID
		})

		providers[providerID] = provider
	}

	return Snapshot{
		Version:   SnapshotVersion,
		SavedAt:   now,
		Providers: providers,
	}
}
