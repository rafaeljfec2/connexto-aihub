package catalog_test

import (
	"reflect"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

func TestDeriveViewListingMatrix(t *testing.T) {
	now := testNow()
	staleAfter := 24 * time.Hour
	fresh := now.Add(-time.Hour)
	stale := now.Add(-25 * time.Hour)

	present := testEntry()
	missing := testEntry()
	missing.MissingSince = testTimePtr(now.Add(-2 * time.Hour))

	cases := []struct {
		name          string
		entry         catalog.ModelEntry
		sync          catalog.ProviderSyncState
		providerState providers.State
		wantListing   catalog.ListingState
		wantPresence  catalog.Presence
		wantFreshness catalog.Freshness
		wantReasons   []catalog.ListingReason
	}{
		{
			name:          "fresh sync and healthy provider",
			entry:         present,
			sync:          testSyncState(testTimePtr(fresh), catalog.SyncOutcomeSuccess),
			providerState: providers.StateHealthy,
			wantListing:   catalog.ListingCurrent,
			wantPresence:  catalog.PresencePresent,
			wantFreshness: catalog.FreshnessFresh,
			wantReasons:   []catalog.ListingReason{},
		},
		{
			name:          "stale catalog",
			entry:         present,
			sync:          testSyncState(testTimePtr(stale), catalog.SyncOutcomeSuccess),
			providerState: providers.StateHealthy,
			wantListing:   catalog.ListingStale,
			wantPresence:  catalog.PresencePresent,
			wantFreshness: catalog.FreshnessStale,
			wantReasons:   []catalog.ListingReason{catalog.ReasonCatalogStale},
		},
		{
			name:          "provider disabled",
			entry:         present,
			sync:          testSyncState(testTimePtr(fresh), catalog.SyncOutcomeSuccess),
			providerState: providers.StateDisabled,
			wantListing:   catalog.ListingStale,
			wantPresence:  catalog.PresencePresent,
			wantFreshness: catalog.FreshnessFresh,
			wantReasons:   []catalog.ListingReason{catalog.ReasonProviderDisabled},
		},
		{
			name:          "provider unhealthy",
			entry:         present,
			sync:          testSyncState(testTimePtr(fresh), catalog.SyncOutcomeSuccess),
			providerState: providers.StateUnhealthy,
			wantListing:   catalog.ListingStale,
			wantPresence:  catalog.PresencePresent,
			wantFreshness: catalog.FreshnessFresh,
			wantReasons:   []catalog.ListingReason{catalog.ReasonProviderUnhealthy},
		},
		{
			name:          "provider state unknown",
			entry:         present,
			sync:          testSyncState(testTimePtr(fresh), catalog.SyncOutcomeSuccess),
			providerState: providers.StateUnknown,
			wantListing:   catalog.ListingStale,
			wantPresence:  catalog.PresencePresent,
			wantFreshness: catalog.FreshnessFresh,
			wantReasons:   []catalog.ListingReason{catalog.ReasonProviderStateUnknown},
		},
		{
			name:          "model missing",
			entry:         missing,
			sync:          testSyncState(testTimePtr(fresh), catalog.SyncOutcomeSuccess),
			providerState: providers.StateHealthy,
			wantListing:   catalog.ListingMissing,
			wantPresence:  catalog.PresenceMissing,
			wantFreshness: catalog.FreshnessFresh,
			wantReasons:   []catalog.ListingReason{catalog.ReasonModelMissing},
		},
		{
			name:          "never synchronized",
			entry:         present,
			sync:          testSyncState(nil, catalog.SyncOutcomeNever),
			providerState: providers.StateHealthy,
			wantListing:   catalog.ListingUnknown,
			wantPresence:  catalog.PresenceUnknown,
			wantFreshness: catalog.FreshnessNever,
			wantReasons:   []catalog.ListingReason{catalog.ReasonNeverSynced},
		},
		{
			name:          "last attempt failed",
			entry:         present,
			sync:          testSyncState(testTimePtr(fresh), catalog.SyncOutcomeFailed),
			providerState: providers.StateHealthy,
			wantListing:   catalog.ListingCurrent,
			wantPresence:  catalog.PresencePresent,
			wantFreshness: catalog.FreshnessFresh,
			wantReasons:   []catalog.ListingReason{catalog.ReasonLastSyncFailed},
		},
		{
			name:          "last attempt rejected empty",
			entry:         present,
			sync:          testSyncState(testTimePtr(fresh), catalog.SyncOutcomeRejectedEmpty),
			providerState: providers.StateHealthy,
			wantListing:   catalog.ListingCurrent,
			wantPresence:  catalog.PresencePresent,
			wantFreshness: catalog.FreshnessFresh,
			wantReasons:   []catalog.ListingReason{catalog.ReasonLastSyncRejectedEmpty},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			view := catalog.DeriveView(testCase.entry, testCase.sync, testCase.providerState, now, staleAfter)

			if !reflect.DeepEqual(view.Entry, testCase.entry) {
				t.Fatalf("view entry = %+v, want %+v", view.Entry, testCase.entry)
			}

			if view.ListingState != testCase.wantListing {
				t.Fatalf("listing state = %q, want %q", view.ListingState, testCase.wantListing)
			}

			if view.Presence != testCase.wantPresence {
				t.Fatalf("presence = %q, want %q", view.Presence, testCase.wantPresence)
			}

			if view.Freshness != testCase.wantFreshness {
				t.Fatalf("freshness = %q, want %q", view.Freshness, testCase.wantFreshness)
			}

			if view.ProviderState != testCase.providerState {
				t.Fatalf("provider state = %q, want %q", view.ProviderState, testCase.providerState)
			}

			if view.ExecutionVerification != catalog.VerificationUnverified {
				t.Fatalf("execution verification = %q, want %q", view.ExecutionVerification, catalog.VerificationUnverified)
			}

			if !reflect.DeepEqual(view.ListingReasons, testCase.wantReasons) {
				t.Fatalf("listing reasons = %v, want %v", view.ListingReasons, testCase.wantReasons)
			}
		})
	}
}

func TestDeriveViewReasonOrder(t *testing.T) {
	now := testNow()
	staleAfter := 24 * time.Hour
	stale := now.Add(-25 * time.Hour)

	missing := testEntry()
	missing.MissingSince = testTimePtr(now.Add(-time.Hour))

	cases := []struct {
		name          string
		entry         catalog.ModelEntry
		sync          catalog.ProviderSyncState
		providerState providers.State
		wantReasons   []catalog.ListingReason
	}{
		{
			name:          "missing and unhealthy",
			entry:         missing,
			sync:          testSyncState(testTimePtr(now.Add(-time.Hour)), catalog.SyncOutcomeSuccess),
			providerState: providers.StateUnhealthy,
			wantReasons:   []catalog.ListingReason{catalog.ReasonModelMissing, catalog.ReasonProviderUnhealthy},
		},
		{
			name:          "never synced and disabled",
			entry:         testEntry(),
			sync:          testSyncState(nil, catalog.SyncOutcomeNever),
			providerState: providers.StateDisabled,
			wantReasons:   []catalog.ListingReason{catalog.ReasonNeverSynced, catalog.ReasonProviderDisabled},
		},
		{
			name:          "missing and stale",
			entry:         missing,
			sync:          testSyncState(testTimePtr(stale), catalog.SyncOutcomeSuccess),
			providerState: providers.StateHealthy,
			wantReasons:   []catalog.ListingReason{catalog.ReasonModelMissing, catalog.ReasonCatalogStale},
		},
		{
			name:          "stale, unhealthy and failed",
			entry:         testEntry(),
			sync:          testSyncState(testTimePtr(stale), catalog.SyncOutcomeFailed),
			providerState: providers.StateUnhealthy,
			wantReasons:   []catalog.ListingReason{catalog.ReasonCatalogStale, catalog.ReasonProviderUnhealthy, catalog.ReasonLastSyncFailed},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			view := catalog.DeriveView(testCase.entry, testCase.sync, testCase.providerState, now, staleAfter)

			if !reflect.DeepEqual(view.ListingReasons, testCase.wantReasons) {
				t.Fatalf("listing reasons = %v, want %v", view.ListingReasons, testCase.wantReasons)
			}
		})
	}
}

func TestDeriveViewFreshnessBoundary(t *testing.T) {
	now := testNow()
	staleAfter := 24 * time.Hour

	cases := []struct {
		name          string
		lastSuccess   time.Time
		wantFreshness catalog.Freshness
	}{
		{
			name:          "exactly at threshold is fresh",
			lastSuccess:   now.Add(-staleAfter),
			wantFreshness: catalog.FreshnessFresh,
		},
		{
			name:          "one nanosecond past threshold is stale",
			lastSuccess:   now.Add(-staleAfter - time.Nanosecond),
			wantFreshness: catalog.FreshnessStale,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			view := catalog.DeriveView(
				testEntry(),
				testSyncState(testTimePtr(testCase.lastSuccess), catalog.SyncOutcomeSuccess),
				providers.StateHealthy,
				now,
				staleAfter,
			)

			if view.Freshness != testCase.wantFreshness {
				t.Fatalf("freshness = %q, want %q", view.Freshness, testCase.wantFreshness)
			}
		})
	}
}

func TestDeriveViewNeverSyncedTakesPrecedenceOverMissing(t *testing.T) {
	now := testNow()
	entry := testEntry()
	entry.MissingSince = testTimePtr(now.Add(-time.Hour))

	view := catalog.DeriveView(entry, testSyncState(nil, catalog.SyncOutcomeNever), providers.StateHealthy, now, 24*time.Hour)

	if view.Presence != catalog.PresenceMissing {
		t.Fatalf("presence = %q, want %q", view.Presence, catalog.PresenceMissing)
	}

	if view.Freshness != catalog.FreshnessNever {
		t.Fatalf("freshness = %q, want %q", view.Freshness, catalog.FreshnessNever)
	}

	if view.ListingState != catalog.ListingUnknown {
		t.Fatalf("listing state = %q, want %q", view.ListingState, catalog.ListingUnknown)
	}

	wantReasons := []catalog.ListingReason{catalog.ReasonNeverSynced, catalog.ReasonModelMissing}
	if !reflect.DeepEqual(view.ListingReasons, wantReasons) {
		t.Fatalf("listing reasons = %v, want %v", view.ListingReasons, wantReasons)
	}
}
