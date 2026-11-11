package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"aihub/internal/providers"
)

func managerBoolPtr(value bool) *bool {
	return &value
}

func managerInt64Ptr(value int64) *int64 {
	return &value
}

func managerDescriptor(id string, image, tools bool) providers.ModelDescriptor {
	input := []string{"text"}
	if image {
		input = append(input, "image/png")
	}

	return providers.ModelDescriptor{
		ID:                      id,
		Name:                    id,
		InputModalities:         input,
		InputModalitiesPresent:  true,
		OutputModalities:        []string{"text"},
		OutputModalitiesPresent: true,
		ToolsSupported:          managerBoolPtr(tools),
	}
}

func managerStatuses(state providers.State) func() []providers.Status {
	return func() []providers.Status {
		return []providers.Status{{
			ID:      providers.OpenCodeGoID,
			Enabled: state != providers.StateDisabled,
			State:   state,
		}}
	}
}

func managerOptions(now time.Time) Options {
	return Options{
		StaleAfter:  24 * time.Hour,
		SyncTimeout: time.Second,
		Now:         func() time.Time { return now },
	}
}

func newTestManagerWithStore(t *testing.T, source SourceFunc, state providers.State, options Options) (*Manager, *Store) {
	t.Helper()

	store := storeTestStore(t.TempDir(), osOps{})
	manager := NewManager(storeTestLogger(), managerStatuses(state), map[string]SourceFunc{
		providers.OpenCodeGoID: source,
	}, store, options)

	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	return manager, store
}

func newTestManager(t *testing.T, source SourceFunc, state providers.State, options Options) *Manager {
	t.Helper()

	manager, _ := newTestManagerWithStore(t, source, state, options)

	return manager
}

func waitForOutcome(t *testing.T, manager *Manager, want SyncOutcome) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		manager.mu.RLock()
		state := manager.syncStates[providers.OpenCodeGoID]
		manager.mu.RUnlock()

		if state.Outcome == want {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatalf("sync outcome did not become %q", want)
}

func TestManagerSyncSuccessCountersAndMissing(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	var current []providers.ModelDescriptor

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))
	ctx := context.Background()

	current = []providers.ModelDescriptor{managerDescriptor("A", true, true), managerDescriptor("B", false, false)}

	result, err := manager.Sync(ctx, providers.OpenCodeGoID)
	if err != nil {
		t.Fatalf("first Sync() error = %v", err)
	}

	if result.Counts != (SyncCounts{Discovered: 2, Added: 2}) {
		t.Fatalf("first counts = %+v", result.Counts)
	}

	current = []providers.ModelDescriptor{managerDescriptor("A", true, true), managerDescriptor("B", false, false)}

	result, err = manager.Sync(ctx, providers.OpenCodeGoID)
	if err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}

	if result.Counts != (SyncCounts{Discovered: 2, Unchanged: 2}) {
		t.Fatalf("second counts = %+v", result.Counts)
	}

	current = []providers.ModelDescriptor{managerDescriptor("A", true, true)}

	result, err = manager.Sync(ctx, providers.OpenCodeGoID)
	if err != nil {
		t.Fatalf("third Sync() error = %v", err)
	}

	if result.Counts != (SyncCounts{Discovered: 1, Unchanged: 1, Missing: 1}) {
		t.Fatalf("third counts = %+v", result.Counts)
	}

	if _, ok := manager.Get(providers.OpenCodeGoID, "B"); !ok {
		t.Fatal("missing model B was deleted")
	}

	manager.mu.RLock()
	missingEntry := manager.entries[ModelKey{ProviderID: providers.OpenCodeGoID, ModelID: "B"}]
	manager.mu.RUnlock()

	if missingEntry.MissingSince == nil {
		t.Fatal("model B missing_since = nil, want set")
	}

	current = []providers.ModelDescriptor{managerDescriptor("A", true, true), managerDescriptor("B", false, false)}

	result, err = manager.Sync(ctx, providers.OpenCodeGoID)
	if err != nil {
		t.Fatalf("fourth Sync() error = %v", err)
	}

	if result.Counts != (SyncCounts{Discovered: 2, Updated: 1, Unchanged: 1}) {
		t.Fatalf("fourth counts = %+v", result.Counts)
	}

	if result.Counts.Discovered != result.Counts.Added+result.Counts.Updated+result.Counts.Unchanged {
		t.Fatalf("counter invariant violated: %+v", result.Counts)
	}

	manager.mu.RLock()
	reappeared := manager.entries[ModelKey{ProviderID: providers.OpenCodeGoID, ModelID: "B"}]
	manager.mu.RUnlock()

	if reappeared.MissingSince != nil {
		t.Fatal("model B missing_since != nil after reappearance")
	}
}

func TestManagerRejectedEmptyKeepsAcceptedState(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	var current []providers.ModelDescriptor

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))
	ctx := context.Background()

	current = []providers.ModelDescriptor{managerDescriptor("A", false, false)}

	first, err := manager.Sync(ctx, providers.OpenCodeGoID)
	if err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	acceptedCounts := first.Counts
	acceptedState, _ := manager.ProviderSyncState(providers.OpenCodeGoID)

	current = nil

	result, err := manager.Sync(ctx, providers.OpenCodeGoID)
	if err != nil {
		t.Fatalf("empty Sync() error = %v", err)
	}

	if result.Outcome != SyncOutcomeRejectedEmpty {
		t.Fatalf("outcome = %q, want rejected_empty", result.Outcome)
	}

	if result.Counts != acceptedCounts {
		t.Fatalf("counts = %+v, want %+v", result.Counts, acceptedCounts)
	}

	if _, ok := manager.Get(providers.OpenCodeGoID, "A"); !ok {
		t.Fatal("accepted model was removed by rejected_empty")
	}

	state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
	if state.LastSuccessAt == nil || acceptedState.LastSuccessAt == nil || !state.LastSuccessAt.Equal(*acceptedState.LastSuccessAt) {
		t.Fatal("last_success_at changed on rejected_empty")
	}

	if state.ErrorClass != SyncErrorEmptyCatalog {
		t.Fatalf("error class = %q, want empty_catalog", state.ErrorClass)
	}
}

func TestManagerClassifiesUpstreamFailures(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	var current []providers.ModelDescriptor
	var currentErr error

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, currentErr
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))
	ctx := context.Background()

	current = []providers.ModelDescriptor{managerDescriptor("A", false, false)}

	if _, err := manager.Sync(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	cases := []struct {
		name       string
		err        error
		wantClass  SyncErrorClass
		wantStatus int
	}{
		{name: "http status", err: &providers.UpstreamHTTPError{StatusCode: 503}, wantClass: SyncErrorHTTPStatus, wantStatus: 503},
		{name: "decode", err: &providers.DecodeError{Err: errors.New("bad")}, wantClass: SyncErrorInvalidPayload},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			currentErr = testCase.err

			_, err := manager.Sync(ctx, providers.OpenCodeGoID)
			if err == nil {
				t.Fatal("Sync() error = nil, want upstream error")
			}

			state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
			if state.Outcome != SyncOutcomeFailed || state.ErrorClass != testCase.wantClass {
				t.Fatalf("state = %+v, want failed/%s", state, testCase.wantClass)
			}

			if testCase.wantStatus != 0 && state.UpstreamStatus != testCase.wantStatus {
				t.Fatalf("upstream status = %d, want %d", state.UpstreamStatus, testCase.wantStatus)
			}

			if _, ok := manager.Get(providers.OpenCodeGoID, "A"); !ok {
				t.Fatal("accepted model was removed by a failed sync")
			}
		})
	}
}

func TestManagerTimeoutIsClassified(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)

	source := func(ctx context.Context) ([]providers.ModelDescriptor, error) {
		<-ctx.Done()

		return nil, ctx.Err()
	}

	// The sync deadline also bounds the snapshot save. A budget far above a
	// local save keeps the test about classification, not disk latency under
	// -race and parallel package load.
	options := managerOptions(now)
	options.SyncTimeout = 250 * time.Millisecond

	manager := newTestManager(t, source, providers.StateHealthy, options)

	_, err := manager.Sync(context.Background(), providers.OpenCodeGoID)
	if err == nil {
		t.Fatal("Sync() error = nil, want timeout")
	}

	state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
	if state.Outcome != SyncOutcomeFailed || state.ErrorClass != SyncErrorTimeout {
		t.Fatalf("state = %+v, want failed/timeout", state)
	}
}

func TestManagerPersistenceFailureKeepsAcceptedState(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	var current []providers.ModelDescriptor

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}

	manager, store := newTestManagerWithStore(t, source, providers.StateHealthy, managerOptions(now))
	ctx := context.Background()

	current = []providers.ModelDescriptor{managerDescriptor("A", false, false)}

	if _, err := manager.Sync(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	acceptedState, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
	store.ops = &failingOps{base: osOps{}, renameErr: errors.New("rename denied")}

	current = []providers.ModelDescriptor{managerDescriptor("A", false, false), managerDescriptor("B", false, false)}

	_, err := manager.Sync(ctx, providers.OpenCodeGoID)
	if !errors.Is(err, ErrPersistence) {
		t.Fatalf("Sync() error = %v, want ErrPersistence", err)
	}

	if _, ok := manager.Get(providers.OpenCodeGoID, "B"); ok {
		t.Fatal("candidate model B was published despite persistence failure")
	}

	state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
	if state.Outcome != SyncOutcomeFailed || state.ErrorClass != SyncErrorPersistence {
		t.Fatalf("state = %+v, want failed/persistence", state)
	}

	if state.Counts != acceptedState.Counts {
		t.Fatalf("counts = %+v, want accepted %+v", state.Counts, acceptedState.Counts)
	}

	if state.LastSuccessAt == nil || acceptedState.LastSuccessAt == nil || !state.LastSuccessAt.Equal(*acceptedState.LastSuccessAt) {
		t.Fatal("last_success_at changed on persistence failure")
	}
}

func TestManagerSingleFlight(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	started := make(chan struct{})
	release := make(chan struct{})

	source := func(ctx context.Context) ([]providers.ModelDescriptor, error) {
		close(started)
		<-release

		return []providers.ModelDescriptor{managerDescriptor("A", false, false)}, nil
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))

	done := make(chan error, 1)

	go func() {
		_, err := manager.Sync(context.Background(), providers.OpenCodeGoID)
		done <- err
	}()

	<-started

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); !errors.Is(err, ErrSyncInProgress) {
		t.Fatalf("second Sync() error = %v, want ErrSyncInProgress", err)
	}

	close(release)

	if err := <-done; err != nil {
		t.Fatalf("first Sync() error = %v", err)
	}
}

func TestManagerRejectsDisabledUnknownAndStopped(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return nil, nil
	}

	disabled := newTestManager(t, source, providers.StateDisabled, managerOptions(now))
	if _, err := disabled.Sync(context.Background(), providers.OpenCodeGoID); !errors.Is(err, ErrProviderDisabled) {
		t.Fatalf("disabled Sync() error = %v, want ErrProviderDisabled", err)
	}

	if _, err := disabled.Sync(context.Background(), "other"); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("unknown provider Sync() error = %v, want ErrProviderNotFound", err)
	}

	stopped := newTestManager(t, source, providers.StateHealthy, managerOptions(now))
	if err := stopped.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if _, err := stopped.Sync(context.Background(), providers.OpenCodeGoID); !errors.Is(err, ErrStopped) {
		t.Fatalf("stopped Sync() error = %v, want ErrStopped", err)
	}
}

func TestManagerStartSkipsAndSyncs(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)

	t.Run("disabled does not fetch", func(t *testing.T) {
		calls := 0
		source := func(context.Context) ([]providers.ModelDescriptor, error) {
			calls++

			return []providers.ModelDescriptor{managerDescriptor("A", false, false)}, nil
		}

		manager := newTestManager(t, source, providers.StateDisabled, managerOptions(now))

		if err := manager.Start(context.Background()); err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
		if state.Outcome != SyncOutcomeSkipped || state.ErrorClass != SyncErrorProviderDisabled {
			t.Fatalf("state = %+v, want skipped/provider_disabled", state)
		}

		if calls != 0 {
			t.Fatalf("source calls = %d, want 0", calls)
		}
	})

	t.Run("unhealthy skips", func(t *testing.T) {
		source := func(context.Context) ([]providers.ModelDescriptor, error) {
			return nil, nil
		}

		manager := newTestManager(t, source, providers.StateUnhealthy, managerOptions(now))

		if err := manager.Start(context.Background()); err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
		if state.Outcome != SyncOutcomeSkipped || state.ErrorClass != SyncErrorProviderUnhealthy {
			t.Fatalf("state = %+v, want skipped/provider_unhealthy", state)
		}
	})

	t.Run("unknown skips without error class", func(t *testing.T) {
		source := func(context.Context) ([]providers.ModelDescriptor, error) {
			return nil, nil
		}

		manager := newTestManager(t, source, providers.StateUnknown, managerOptions(now))

		if err := manager.Start(context.Background()); err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
		if state.Outcome != SyncOutcomeSkipped || state.ErrorClass != SyncErrorNone {
			t.Fatalf("state = %+v, want skipped/none", state)
		}
	})

	t.Run("healthy syncs asynchronously", func(t *testing.T) {
		source := func(context.Context) ([]providers.ModelDescriptor, error) {
			return []providers.ModelDescriptor{managerDescriptor("A", false, false)}, nil
		}

		manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))

		if err := manager.Start(context.Background()); err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		waitForOutcome(t, manager, SyncOutcomeSuccess)

		if _, ok := manager.Get(providers.OpenCodeGoID, "A"); !ok {
			t.Fatal("startup sync did not publish model A")
		}
	})
}

func TestManagerStopCancelsStartupSync(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	started := make(chan struct{})
	cancelled := make(chan struct{})

	source := func(ctx context.Context) ([]providers.ModelDescriptor, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)

		return nil, ctx.Err()
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	<-started

	if err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("startup source did not observe cancellation")
	}
}

func TestManagerDurabilityUncertainCommits(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return []providers.ModelDescriptor{managerDescriptor("A", false, false)}, nil
	}

	manager, store := newTestManagerWithStore(t, source, providers.StateHealthy, managerOptions(now))
	store.ops = &failingOps{base: osOps{}, syncDirErr: errors.New("dir sync denied")}

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("Sync() error = %v, want nil for durability-uncertain write", err)
	}

	state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
	if state.Outcome != SyncOutcomeSuccess {
		t.Fatalf("state = %+v, want success", state)
	}

	if _, ok := manager.Get(providers.OpenCodeGoID, "A"); !ok {
		t.Fatal("durability-uncertain sync did not publish model A")
	}
}

func TestManagerReloadsPersistedCatalog(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return []providers.ModelDescriptor{managerDescriptor("A", true, true)}, nil
	}

	manager, store := newTestManagerWithStore(t, source, providers.StateHealthy, managerOptions(now))
	ctx := context.Background()

	if _, err := manager.Sync(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	reloaded := NewManager(storeTestLogger(), managerStatuses(providers.StateDisabled), map[string]SourceFunc{
		providers.OpenCodeGoID: source,
	}, store, managerOptions(now))

	t.Cleanup(func() { _ = reloaded.Stop(context.Background()) })

	if err := reloaded.Start(ctx); err != nil {
		t.Fatalf("reloaded Start() error = %v", err)
	}

	if _, ok := reloaded.Get(providers.OpenCodeGoID, "A"); !ok {
		t.Fatal("persisted model was not reloaded")
	}
}

func TestManagerQueryFilters(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	var current []providers.ModelDescriptor

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))
	ctx := context.Background()

	current = []providers.ModelDescriptor{
		managerDescriptor("A", true, true),
		managerDescriptor("B", false, false),
		managerDescriptor("C", false, true),
	}

	if _, err := manager.Sync(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	current = []providers.ModelDescriptor{
		managerDescriptor("A", true, true),
		managerDescriptor("B", false, false),
	}

	if _, err := manager.Sync(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}

	all := manager.List(Filter{})
	if len(all) != 3 {
		t.Fatalf("List() count = %d, want 3", len(all))
	}

	if all[0].Entry.ModelID != "A" || all[1].Entry.ModelID != "B" || all[2].Entry.ModelID != "C" {
		t.Fatalf("List() order = %s/%s/%s", all[0].Entry.ModelID, all[1].Entry.ModelID, all[2].Entry.ModelID)
	}

	missing := PresenceMissing
	missingViews := manager.List(Filter{Presence: &missing})
	if len(missingViews) != 1 || missingViews[0].Entry.ModelID != "C" {
		t.Fatalf("missing filter = %+v", missingViews)
	}

	currentListing := ListingCurrent
	currentViews := manager.List(Filter{ListingState: &currentListing})
	if len(currentViews) != 2 {
		t.Fatalf("current filter count = %d, want 2", len(currentViews))
	}

	imageViews := manager.List(Filter{Capability: "image_input"})
	if len(imageViews) != 1 || imageViews[0].Entry.ModelID != "A" {
		t.Fatalf("image filter = %+v", imageViews)
	}

	if _, ok := manager.Get(providers.OpenCodeGoID, "A"); !ok {
		t.Fatal("Get(A) not found")
	}

	if _, ok := manager.Get(providers.OpenCodeGoID, "missing"); ok {
		t.Fatal("Get(missing) found")
	}

	providersViews := manager.Providers()
	if len(providersViews) != 1 {
		t.Fatalf("Providers() count = %d, want 1", len(providersViews))
	}

	if providersViews[0].ModelsKnown != 3 || providersViews[0].ModelsPresent != 2 || providersViews[0].ModelsMissing != 1 {
		t.Fatalf("provider counts = %+v", providersViews[0])
	}
}

func TestManagerSkipPersistenceFailureKeepsPreviousState(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return nil, nil
	}

	manager, store := newTestManagerWithStore(t, source, providers.StateDisabled, managerOptions(now))
	store.ops = &failingOps{base: osOps{}, renameErr: errors.New("rename denied")}

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
	if state.Outcome != SyncOutcomeNever || state.LastAttemptAt != nil {
		t.Fatalf("state = %+v, want untouched never state", state)
	}
}

func TestManagerPersistenceFailureMetadataLostOnRestart(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	var current []providers.ModelDescriptor

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}

	manager, store := newTestManagerWithStore(t, source, providers.StateHealthy, managerOptions(now))
	ctx := context.Background()

	current = []providers.ModelDescriptor{managerDescriptor("A", false, false)}
	if _, err := manager.Sync(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	store.ops = &failingOps{base: osOps{}, renameErr: errors.New("rename denied")}
	current = []providers.ModelDescriptor{managerDescriptor("A", false, false), managerDescriptor("B", false, false)}

	if _, err := manager.Sync(ctx, providers.OpenCodeGoID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Sync() error = %v, want ErrPersistence", err)
	}

	snapshot, _ := store.Load(ctx)
	provider := snapshot.Providers[providers.OpenCodeGoID]

	if provider.Sync.Outcome != SyncOutcomeSuccess {
		t.Fatalf("persisted outcome = %q, want success", provider.Sync.Outcome)
	}

	if len(provider.Models) != 1 || provider.Models[0].ModelID != "A" {
		t.Fatalf("persisted models = %+v, want only A", provider.Models)
	}
}

func TestManagerCallerCancellationDiscardsCandidate(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	started := make(chan struct{})
	release := make(chan struct{})

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		close(started)
		<-release

		return []providers.ModelDescriptor{managerDescriptor("A", false, false)}, nil
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		_, err := manager.Sync(ctx, providers.OpenCodeGoID)
		done <- err
	}()

	<-started
	cancel()
	close(release)

	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Sync() error = %v, want context.Canceled", err)
	}

	if _, ok := manager.Get(providers.OpenCodeGoID, "A"); ok {
		t.Fatal("cancelled sync published model A")
	}

	state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
	if state.Outcome != SyncOutcomeNever {
		t.Fatalf("state = %+v, want never", state)
	}
}

func TestManagerDecodeErrorKeepsObservedStatus(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return nil, &providers.DecodeError{StatusCode: 200, Err: errors.New("bad payload")}
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err == nil {
		t.Fatal("Sync() error = nil, want decode error")
	}

	state, _ := manager.ProviderSyncState(providers.OpenCodeGoID)
	if state.ErrorClass != SyncErrorInvalidPayload || state.UpstreamStatus != 200 {
		t.Fatalf("state = %+v, want invalid_payload/200", state)
	}
}

func TestManagerQueryViewsAreIsolated(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	descriptor := managerDescriptor("A", true, true)
	descriptor.LimitContext = managerInt64Ptr(1000)

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return []providers.ModelDescriptor{descriptor}, nil
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))

	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	view, ok := manager.Get(providers.OpenCodeGoID, "A")
	if !ok {
		t.Fatal("Get(A) not found")
	}

	view.Entry.Capabilities["text_generation"] = CapabilityClaim{State: CapabilityUnsupported}
	view.Entry.Modalities.Input[0] = "mutated"
	*view.Entry.Limits.Context = 1

	again, ok := manager.Get(providers.OpenCodeGoID, "A")
	if !ok {
		t.Fatal("second Get(A) not found")
	}

	if again.Entry.Capabilities["text_generation"].State != CapabilitySupported {
		t.Fatal("capabilities map was shared with the caller")
	}

	if again.Entry.Modalities.Input[0] != "text" {
		t.Fatal("modalities slice was shared with the caller")
	}

	if again.Entry.Limits.Context == nil || *again.Entry.Limits.Context != 1000 {
		t.Fatalf("limits pointer was shared with the caller: %v", again.Entry.Limits.Context)
	}
}

func TestManagerRawMetadataReorderingIsNotAnUpdate(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	first := managerDescriptor("A", false, false)
	first.Raw = json.RawMessage(`{"a":1,"b":2}`)
	second := managerDescriptor("A", false, false)
	second.Raw = json.RawMessage(`{"b":2,"a":1}`)

	var current []providers.ModelDescriptor
	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))
	ctx := context.Background()

	current = []providers.ModelDescriptor{first}
	if _, err := manager.Sync(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	current = []providers.ModelDescriptor{second}

	result, err := manager.Sync(ctx, providers.OpenCodeGoID)
	if err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}

	if result.Counts.Unchanged != 1 || result.Counts.Updated != 0 {
		t.Fatalf("counts = %+v, want unchanged=1 updated=0", result.Counts)
	}

	view, ok := manager.Get(providers.OpenCodeGoID, "A")
	if !ok {
		t.Fatal("Get(A) not found")
	}

	if string(view.Entry.ProviderMetadata) != `{"b":2,"a":1}` {
		t.Fatalf("raw metadata = %s, want refreshed raw", view.Entry.ProviderMetadata)
	}
}

func TestManagerQueryMissingSinceIsIsolated(t *testing.T) {
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	var current []providers.ModelDescriptor

	source := func(context.Context) ([]providers.ModelDescriptor, error) {
		return current, nil
	}

	manager := newTestManager(t, source, providers.StateHealthy, managerOptions(now))

	current = []providers.ModelDescriptor{managerDescriptor("A", false, false), managerDescriptor("B", false, false)}
	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("seed Sync() error = %v", err)
	}

	current = []providers.ModelDescriptor{managerDescriptor("A", false, false)}
	if _, err := manager.Sync(context.Background(), providers.OpenCodeGoID); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}

	view, ok := manager.Get(providers.OpenCodeGoID, "B")
	if !ok || view.Entry.MissingSince == nil {
		t.Fatal("missing model B has no missing_since")
	}

	original := *view.Entry.MissingSince
	*view.Entry.MissingSince = now.Add(24 * time.Hour)

	again, ok := manager.Get(providers.OpenCodeGoID, "B")
	if !ok || again.Entry.MissingSince == nil {
		t.Fatal("second Get(B) has no missing_since")
	}

	if !again.Entry.MissingSince.Equal(original) {
		t.Fatalf("missing_since pointer was shared with the caller: %v", again.Entry.MissingSince)
	}
}

func TestManagerStartDropsLegacyTransportCatalogProvider(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()
	store := storeTestStore(t.TempDir(), osOps{})

	legacyEntry := NormalizeEntry(providers.ConnextoProxyID, providers.ModelDescriptor{
		ID:   "composer-2.5",
		Name: "composer-2.5",
	}, now)
	cursorEntry := NormalizeEntry(providers.CursorID, providers.ModelDescriptor{
		ID:   "composer-2.5",
		Name: "composer-2.5",
	}, now)

	snapshot := Snapshot{
		Version: SnapshotVersion,
		SavedAt: now,
		Providers: map[string]ProviderSnapshot{
			providers.ConnextoProxyID: {
				Sync:   ProviderSyncState{Outcome: SyncOutcomeSuccess, LastSuccessAt: &now},
				Models: []ModelEntry{legacyEntry},
			},
			providers.CursorID: {
				Sync:   ProviderSyncState{Outcome: SyncOutcomeSuccess, LastSuccessAt: &now},
				Models: []ModelEntry{cursorEntry},
			},
		},
	}
	if _, err := store.Save(ctx, snapshot); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	manager := NewManager(storeTestLogger(), func() []providers.Status {
		return []providers.Status{
			{ID: providers.CursorID, Enabled: true, State: providers.StateHealthy},
		}
	}, map[string]SourceFunc{
		providers.CursorID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return nil, errors.New("not called in this test")
		},
	}, store, managerOptions(now))
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	for _, view := range manager.List(Filter{}) {
		if providers.ExcludedFromCatalog(view.Entry.ProviderID) {
			t.Fatalf("List() still returned transport-only provider %q", view.Entry.ProviderID)
		}
	}

	reloaded, _ := store.Load(ctx)
	if _, ok := reloaded.Providers[providers.ConnextoProxyID]; ok {
		t.Fatal("persisted snapshot still contains legacy transport provider")
	}
}
