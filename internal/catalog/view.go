package catalog

import (
	"time"

	"aihub/internal/providers"
)

// DeriveView combines persisted catalog facts with live provider state into a
// read-time view. The result is advisory: listing_state is a catalog condition,
// never proof that a model can execute a request.
func DeriveView(entry ModelEntry, sync ProviderSyncState, providerState providers.State, now time.Time, staleAfter time.Duration) View {
	presence := derivePresence(entry, sync)
	freshness := deriveFreshness(sync, now, staleAfter)

	return View{
		Entry:                 entry,
		Presence:              presence,
		Freshness:             freshness,
		ProviderState:         providerState,
		ExecutionVerification: VerificationUnverified,
		ListingState:          deriveListingState(presence, freshness, providerState),
		ListingReasons:        deriveListingReasons(presence, freshness, providerState, sync.Outcome),
	}
}

func derivePresence(entry ModelEntry, sync ProviderSyncState) Presence {
	if entry.MissingSince != nil {
		return PresenceMissing
	}

	if sync.LastSuccessAt == nil {
		return PresenceUnknown
	}

	return PresencePresent
}

func deriveFreshness(sync ProviderSyncState, now time.Time, staleAfter time.Duration) Freshness {
	if sync.LastSuccessAt == nil {
		return FreshnessNever
	}

	if now.Sub(*sync.LastSuccessAt) <= staleAfter {
		return FreshnessFresh
	}

	return FreshnessStale
}

func deriveListingState(presence Presence, freshness Freshness, providerState providers.State) ListingState {
	if freshness == FreshnessNever {
		return ListingUnknown
	}

	if presence == PresenceMissing {
		return ListingMissing
	}

	if presence == PresencePresent && freshness == FreshnessFresh && providerState == providers.StateHealthy {
		return ListingCurrent
	}

	return ListingStale
}

func deriveListingReasons(presence Presence, freshness Freshness, providerState providers.State, outcome SyncOutcome) []ListingReason {
	reasons := make([]ListingReason, 0, 4)

	if freshness == FreshnessNever {
		reasons = append(reasons, ReasonNeverSynced)
	}

	if presence == PresenceMissing {
		reasons = append(reasons, ReasonModelMissing)
	}

	if freshness == FreshnessStale {
		reasons = append(reasons, ReasonCatalogStale)
	}

	switch providerState {
	case providers.StateDisabled:
		reasons = append(reasons, ReasonProviderDisabled)
	case providers.StateUnhealthy:
		reasons = append(reasons, ReasonProviderUnhealthy)
	case providers.StateUnknown:
		reasons = append(reasons, ReasonProviderStateUnknown)
	case providers.StateHealthy:
	}

	switch outcome {
	case SyncOutcomeFailed:
		reasons = append(reasons, ReasonLastSyncFailed)
	case SyncOutcomeRejectedEmpty:
		reasons = append(reasons, ReasonLastSyncRejectedEmpty)
	case SyncOutcomeNever, SyncOutcomeSuccess, SyncOutcomeSkipped:
	}

	return reasons
}
