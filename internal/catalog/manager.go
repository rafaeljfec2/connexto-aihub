package catalog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync"
	"time"

	"aihub/internal/providers"
)

// SourceFunc fetches provider descriptors. It is a function, not an interface,
// because E03 has exactly one provider.
type SourceFunc func(context.Context) ([]providers.ModelDescriptor, error)

// Options tunes the catalog manager.
type Options struct {
	StaleAfter  time.Duration
	SyncTimeout time.Duration
	Now         func() time.Time
}

// SyncResult reports one manual or startup synchronization.
type SyncResult struct {
	ProviderID string
	Outcome    SyncOutcome
	Counts     SyncCounts
	Duration   time.Duration
}

// Manager owns the catalog state and implements app.Module. It never fails
// startup because of catalog or provider problems.
type Manager struct {
	logger      *slog.Logger
	statuses    func() []providers.Status
	sources     map[string]SourceFunc
	store       SnapshotStore
	staleAfter  time.Duration
	syncTimeout time.Duration
	now         func() time.Time

	mu         sync.RWMutex
	entries    map[ModelKey]ModelEntry
	syncStates map[string]ProviderSyncState
	closed     bool

	syncMu sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager builds the catalog module.
func NewManager(logger *slog.Logger, statuses func() []providers.Status, sources map[string]SourceFunc, store SnapshotStore, opts Options) *Manager {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	if statuses == nil {
		statuses = func() []providers.Status { return nil }
	}

	if sources == nil {
		sources = map[string]SourceFunc{}
	}

	if opts.StaleAfter <= 0 {
		opts.StaleAfter = 24 * time.Hour
	}

	if opts.SyncTimeout <= 0 {
		opts.SyncTimeout = 10 * time.Second
	}

	if opts.Now == nil {
		opts.Now = time.Now
	}

	manager := &Manager{
		logger:      logger,
		statuses:    statuses,
		sources:     sources,
		store:       store,
		staleAfter:  opts.StaleAfter,
		syncTimeout: opts.SyncTimeout,
		now:         opts.Now,
		entries:     make(map[ModelKey]ModelEntry),
		syncStates:  make(map[string]ProviderSyncState),
	}

	for providerID := range sources {
		manager.syncStates[providerID] = ProviderSyncState{Outcome: SyncOutcomeNever, ErrorClass: SyncErrorNone}
	}

	return manager
}

// Name implements app.Module.
func (m *Manager) Name() string {
	return "catalog"
}

// Start loads the snapshot, derives the initial provider states and launches at
// most one asynchronous sync. It returns nil for every runtime condition.
func (m *Manager) Start(ctx context.Context) error {
	if m.store == nil {
		return errors.New("catalog: store is required")
	}

	snapshot, _ := m.store.Load(ctx)

	m.mu.Lock()
	m.entries = make(map[ModelKey]ModelEntry)

	for providerID, provider := range snapshot.Providers {
		m.syncStates[providerID] = provider.Sync

		for _, entry := range provider.Models {
			entry.ProviderID = providerID
			m.entries[ModelKey{ProviderID: providerID, ModelID: entry.ModelID}] = entry
		}
	}

	m.closed = false
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	// Load already logs quarantine and writes-disabled diagnostics; the
	// manager only needs the resulting state.

	for _, providerID := range m.ProviderIDs() {
		status := m.statusFor(providerID)

		switch status.State {
		case providers.StateDisabled:
			m.persistSkip(providerID, SyncErrorProviderDisabled)
		case providers.StateHealthy:
			m.launchStartupSync(providerID)
		case providers.StateUnhealthy:
			m.persistSkip(providerID, SyncErrorProviderUnhealthy)
		case providers.StateUnknown:
			m.persistSkip(providerID, SyncErrorNone)
		default:
			m.persistSkip(providerID, SyncErrorNone)
		}
	}

	return nil
}

// Stop cancels and joins the startup sync, waits for an in-flight manual sync
// and marks the manager closed.
func (m *Manager) Stop(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}

	m.wg.Wait()

	m.syncMu.Lock()
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	m.syncMu.Unlock()

	return nil
}

// Sync executes one synchronization attempt and commits it only after the
// snapshot is persisted. A persistence failure publishes no accepted state and
// records the attempt as failed/persistence in memory.
func (m *Manager) Sync(ctx context.Context, providerID string) (SyncResult, error) {
	m.mu.RLock()
	closed := m.closed
	m.mu.RUnlock()

	if closed {
		return SyncResult{}, ErrStopped
	}

	source, ok := m.sources[providerID]
	if !ok {
		return SyncResult{}, ErrProviderNotFound
	}

	if m.statusFor(providerID).State == providers.StateDisabled {
		return SyncResult{}, ErrProviderDisabled
	}

	if !m.syncMu.TryLock() {
		return SyncResult{}, ErrSyncInProgress
	}
	defer m.syncMu.Unlock()

	m.mu.RLock()
	closed = m.closed
	m.mu.RUnlock()

	if closed {
		return SyncResult{}, ErrStopped
	}

	started := m.now()
	syncCtx, cancel := context.WithTimeout(ctx, m.syncTimeout)
	defer cancel()

	descriptors, fetchErr := source(syncCtx)
	now := m.now().UTC()

	var (
		candidateEntries map[ModelKey]ModelEntry
		candidateSync    ProviderSyncState
		counts           SyncCounts
		classifiedErr    error
	)

	switch {
	case fetchErr != nil:
		outcome, class, upstreamStatus := classifySyncError(fetchErr)
		candidateEntries, candidateSync, counts = m.failureCandidate(providerID, outcome, class, upstreamStatus, now)
		classifiedErr = fetchErr
	case len(descriptors) == 0:
		candidateEntries, candidateSync, counts = m.rejectedEmptyCandidate(providerID, now)
	default:
		candidateEntries, candidateSync, counts = m.successCandidate(providerID, descriptors, now)
	}

	if err := ctx.Err(); err != nil {
		m.logger.Warn("catalog synchronization discarded before commit", "provider", providerID, "reason", "context cancelled")

		return SyncResult{
			ProviderID: providerID,
			Outcome:    SyncOutcomeFailed,
			Counts:     m.currentCounts(providerID),
			Duration:   m.now().Sub(started),
		}, err
	}

	persistCtx, cancelPersist := context.WithTimeout(context.Background(), m.syncTimeout)
	defer cancelPersist()

	if _, err := m.persist(persistCtx, providerID, candidateEntries, candidateSync, now); err != nil {
		m.recordPersistenceFailure(providerID, candidateSync.UpstreamStatus, now)
		m.logger.Warn("catalog synchronization not committed; accepted state unchanged",
			"provider", providerID, "error_class", SyncErrorPersistence, "upstream_status", candidateSync.UpstreamStatus)
		acceptedCounts := m.currentCounts(providerID)

		return SyncResult{
			ProviderID: providerID,
			Outcome:    SyncOutcomeFailed,
			Counts:     acceptedCounts,
			Duration:   m.now().Sub(started),
		}, fmt.Errorf("%w: %w", ErrPersistence, err)
	}

	m.commit(providerID, candidateEntries, candidateSync)

	result := SyncResult{
		ProviderID: providerID,
		Outcome:    candidateSync.Outcome,
		Counts:     counts,
		Duration:   m.now().Sub(started),
	}

	if classifiedErr != nil {
		m.logger.Warn("catalog synchronization failed",
			"provider", providerID,
			"outcome", candidateSync.Outcome,
			"error_class", candidateSync.ErrorClass,
			"upstream_status", candidateSync.UpstreamStatus)

		return result, classifiedErr
	}

	return result, nil
}

// ProviderIDs returns the sorted provider ids participating in synchronization.
func (m *Manager) ProviderIDs() []string {
	ids := make([]string, 0, len(m.sources))
	for providerID := range m.sources {
		ids = append(ids, providerID)
	}

	sort.Strings(ids)

	return ids
}

// ProviderSyncState returns a snapshot of one provider's sync state.
func (m *Manager) ProviderSyncState(providerID string) (ProviderSyncState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.syncStates[providerID]

	return state, ok
}

func (m *Manager) launchStartupSync(providerID string) {
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()

		// Sync logs structured outcome/error fields; the raw error (which may
		// contain an upstream URL) is deliberately not logged.
		_, _ = m.Sync(m.ctx, providerID)
	}()
}

func (m *Manager) persistSkip(providerID string, class SyncErrorClass) {
	now := m.now().UTC()

	m.mu.RLock()
	entries := copyEntries(m.entries)
	state := m.syncStates[providerID]
	m.mu.RUnlock()

	state.LastAttemptAt = &now
	state.Outcome = SyncOutcomeSkipped
	state.ErrorClass = class

	if _, err := m.persist(context.Background(), providerID, entries, state, now); err != nil {
		m.logger.Warn("catalog skip state was not persisted", "provider", providerID, "error_class", class)

		return
	}

	m.commit(providerID, entries, state)
}

func (m *Manager) persist(ctx context.Context, providerID string, entries map[ModelKey]ModelEntry, state ProviderSyncState, now time.Time) (SaveResult, error) {
	m.mu.RLock()
	states := make(map[string]ProviderSyncState, len(m.syncStates)+1)

	for id, existing := range m.syncStates {
		states[id] = existing
	}
	m.mu.RUnlock()

	states[providerID] = state

	return m.store.Save(ctx, buildSnapshot(entries, states, now))
}

func (m *Manager) commit(providerID string, entries map[ModelKey]ModelEntry, state ProviderSyncState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = entries
	m.syncStates[providerID] = state
}

func (m *Manager) recordPersistenceFailure(providerID string, upstreamStatus int, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.syncStates[providerID]
	state.Outcome = SyncOutcomeFailed
	state.ErrorClass = SyncErrorPersistence
	state.UpstreamStatus = upstreamStatus
	state.LastAttemptAt = &now
	m.syncStates[providerID] = state
}

func (m *Manager) currentCounts(providerID string) SyncCounts {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.syncStates[providerID].Counts
}

func (m *Manager) statusFor(providerID string) providers.Status {
	for _, status := range m.statuses() {
		if status.ID == providerID {
			return status
		}
	}

	return providers.Status{ID: providerID, State: providers.StateUnknown}
}

func copyEntries(entries map[ModelKey]ModelEntry) map[ModelKey]ModelEntry {
	copied := make(map[ModelKey]ModelEntry, len(entries))

	for key, entry := range entries {
		copied[key] = entry
	}

	return copied
}
