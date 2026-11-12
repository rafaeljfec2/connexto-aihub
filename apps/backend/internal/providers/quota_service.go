package providers

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	accountQueryTimeout = 5 * time.Second
	defaultQuotaStaleTTL = 2 * time.Minute
)

type accountCacheEntry struct {
	snapshots       []ProviderQuotaSnapshot
	lastAttemptAt   time.Time
	lastSuccessAt   time.Time
	lastError       *ProviderQuotaError
	attemptSequence int64
}

// QuotaService orchestrates independent official quota reads for configured accounts.
type QuotaService struct {
	adminService    *Service
	adapters        map[string]QuotaAdapter
	logger          *slog.Logger

	mu              sync.RWMutex
	cache           map[string]accountCacheEntry
	refreshing      map[string]bool
	sf              singleflight.Group
	staleTTL        time.Duration
	sequenceCounter int64

	serviceCtx      context.Context
	serviceCancel   context.CancelFunc
}

// QuotaServiceOption configures QuotaService behavior.
type QuotaServiceOption func(*QuotaService)

// WithQuotaTTL sets the time-to-live before a cached snapshot is considered stale.
func WithQuotaTTL(ttl time.Duration) QuotaServiceOption {
	return func(s *QuotaService) {
		if ttl > 0 {
			s.staleTTL = ttl
		}
	}
}

// NewQuotaService constructs a quota orchestration service.
func NewQuotaService(adminService *Service, adapters map[string]QuotaAdapter, logger *slog.Logger, opts ...QuotaServiceOption) *QuotaService {
	if adapters == nil {
		adapters = make(map[string]QuotaAdapter)
	}
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &QuotaService{
		adminService:  adminService,
		adapters:      adapters,
		logger:        logger,
		cache:         make(map[string]accountCacheEntry),
		refreshing:    make(map[string]bool),
		staleTTL:      defaultQuotaStaleTTL,
		serviceCtx:    ctx,
		serviceCancel: cancel,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Close stops background routines managed by QuotaService.
func (s *QuotaService) Close() {
	if s.serviceCancel != nil {
		s.serviceCancel()
	}
}

// InvalidateAccount removes cached quota state for a deleted provider account.
func (s *QuotaService) InvalidateAccount(accountID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, accountID)
	delete(s.refreshing, accountID)
}

// GetSnapshotsOptions carries optional parameters for snapshot retrieval.
type GetSnapshotsOptions struct {
	ForceRefresh bool
}

// GetSnapshots returns the latest available snapshots immediately without waiting for upstream calls.
// Stale or missing snapshots trigger asynchronous background collection.
func (s *QuotaService) GetSnapshots(ctx context.Context, opts ...GetSnapshotsOptions) (ProviderQuotasResponse, error) {
	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	records, err := s.adminService.List(ctx)
	if err != nil {
		return ProviderQuotasResponse{Snapshots: []ProviderQuotaSnapshot{}, FetchedAt: fetchedAt}, err
	}
	if len(records) == 0 {
		return ProviderQuotasResponse{Snapshots: []ProviderQuotaSnapshot{}, FetchedAt: fetchedAt}, nil
	}

	forceRefresh := false
	if len(opts) > 0 && opts[0].ForceRefresh {
		forceRefresh = true
	}

	snapshots := make([]ProviderQuotaSnapshot, 0, len(records)*2)
	now := time.Now()

	for _, record := range records {
		s.mu.Lock()
		entry, hasCache := s.cache[record.ID]
		isRefreshing := s.refreshing[record.ID]

		needsRefresh := false
		if !hasCache {
			needsRefresh = true
		} else if forceRefresh || now.Sub(entry.lastAttemptAt) > s.staleTTL {
			needsRefresh = true
		}

		if needsRefresh && !isRefreshing {
			s.refreshing[record.ID] = true
			isRefreshing = true
			s.triggerAccountRefresh(record)
		}
		s.mu.Unlock()

		if hasCache && len(entry.snapshots) > 0 {
			for _, snap := range entry.snapshots {
				cp := snap
				if isRefreshing {
					cp.Refreshing = true
				}
				snapshots = append(snapshots, cp)
			}
		} else {
			// Account has never been collected or cache is cold
			initialSnapshots := s.coldStartSnapshots(record, isRefreshing)
			snapshots = append(snapshots, initialSnapshots...)
		}
	}

	return ProviderQuotasResponse{Snapshots: snapshots, FetchedAt: fetchedAt}, nil
}

// QueryAllAccounts synchronously refreshes and returns snapshots across all accounts.
// It preserves existing synchronous behavior, tests, and updates the internal cache.
func (s *QuotaService) QueryAllAccounts(ctx context.Context) (ProviderQuotasResponse, error) {
	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	records, err := s.adminService.List(ctx)
	if err != nil {
		return ProviderQuotasResponse{Snapshots: []ProviderQuotaSnapshot{}, FetchedAt: fetchedAt}, err
	}
	if len(records) == 0 {
		return ProviderQuotasResponse{Snapshots: []ProviderQuotaSnapshot{}, FetchedAt: fetchedAt}, nil
	}

	perAccount := make([][]ProviderQuotaSnapshot, len(records))
	var wait sync.WaitGroup
	for index, record := range records {
		wait.Add(1)
		go func(index int, record Record) {
			defer wait.Done()
			perAccount[index] = s.executeAccountRefresh(ctx, record)
		}(index, record)
	}
	wait.Wait()

	snapshots := make([]ProviderQuotaSnapshot, 0, len(records)*2)
	for _, accountSnapshots := range perAccount {
		snapshots = append(snapshots, accountSnapshots...)
	}
	return ProviderQuotasResponse{Snapshots: snapshots, FetchedAt: fetchedAt}, nil
}

// triggerAccountRefresh launches a background goroutine for a single account using singleflight.
func (s *QuotaService) triggerAccountRefresh(record Record) {
	go func() {
		key := "account:" + record.ID
		_, _, _ = s.sf.Do(key, func() (interface{}, error) {
			defer func() {
				s.mu.Lock()
				delete(s.refreshing, record.ID)
				s.mu.Unlock()
			}()

			timeoutCtx, cancel := context.WithTimeout(s.serviceCtx, accountQueryTimeout)
			if record.Type == TypeCursor {
				cancel()
				timeoutCtx, cancel = context.WithTimeout(s.serviceCtx, cursorAccountQueryTimeout)
			}
			defer cancel()

			return s.executeAccountRefresh(timeoutCtx, record), nil
		})
	}()
}

// executeAccountRefresh queries an account, protects against out-of-order writes,
// and preserves previous valid snapshots on transient failure.
func (s *QuotaService) executeAccountRefresh(ctx context.Context, record Record) []ProviderQuotaSnapshot {
	attemptSeq := atomic.AddInt64(&s.sequenceCounter, 1)
	attemptTime := time.Now()

	freshSnapshots := s.queryConfiguredAccount(ctx, record)

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.cache[record.ID]
	if exists && attemptSeq < existing.attemptSequence {
		// Stale out-of-order write rejection
		return existing.snapshots
	}

	var hasValidFresh bool
	var firstFreshError *ProviderQuotaError
	for _, snap := range freshSnapshots {
		if snap.Status == QuotaStatusAvailable {
			hasValidFresh = true
		}
		if snap.Error != nil && firstFreshError == nil {
			firstFreshError = snap.Error
		}
	}

	nowStr := attemptTime.UTC().Format(time.RFC3339)

	// Preservation policy: if fresh query completely failed or timed out,
	// but we already have valid existing snapshots, retain existing valid metrics.
	if !hasValidFresh && exists && isSnapshotSetDisplayable(existing.snapshots) {
		preservedSnapshots := make([]ProviderQuotaSnapshot, len(existing.snapshots))
		for i, old := range existing.snapshots {
			cp := old
			cp.Stale = true
			cp.Refreshing = false
			cp.LastAttemptAt = &nowStr
			if firstFreshError != nil {
				cp.LastAttemptError = firstFreshError
			}
			preservedSnapshots[i] = cp
		}
		s.cache[record.ID] = accountCacheEntry{
			snapshots:       preservedSnapshots,
			lastAttemptAt:   attemptTime,
			lastSuccessAt:   existing.lastSuccessAt,
			lastError:       firstFreshError,
			attemptSequence: attemptSeq,
		}
		return preservedSnapshots
	}

	// Normal update: stamp metadata and save fresh
	stampedFresh := make([]ProviderQuotaSnapshot, len(freshSnapshots))
	for i, fresh := range freshSnapshots {
		cp := fresh
		cp.Refreshing = false
		cp.LastAttemptAt = &nowStr
		if cp.Status == QuotaStatusAvailable {
			cp.Stale = false
			cp.LastAttemptError = nil
		}
		stampedFresh[i] = cp
	}

	var successTime time.Time
	if hasValidFresh {
		successTime = attemptTime
	} else if exists {
		successTime = existing.lastSuccessAt
	}

	s.cache[record.ID] = accountCacheEntry{
		snapshots:       stampedFresh,
		lastAttemptAt:   attemptTime,
		lastSuccessAt:   successTime,
		lastError:       firstFreshError,
		attemptSequence: attemptSeq,
	}

	return stampedFresh
}

func (s *QuotaService) coldStartSnapshots(record Record, refreshing bool) []ProviderQuotaSnapshot {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	if !record.Enabled {
		return []ProviderQuotaSnapshot{unavailableSnapshot(record, ScopeUnknown, record.Type, QuotaStatusUnsupported,
			"account_disabled", "Conta desabilitada nas configurações")}
	}
	adapter, supported := s.adapters[record.Type]
	if !supported && record.Type != TypeCursor {
		if !record.HasCredential && record.Type != TypeAntigravity {
			return []ProviderQuotaSnapshot{unavailableSnapshot(record, ScopeUnknown, record.Type,
				QuotaStatusUnconfiguredCredential, "unconfigured_credential", "Conta sem credencial configurada")}
		}
		return []ProviderQuotaSnapshot{unavailableSnapshot(record, ScopeUnknown, record.Type, QuotaStatusUnsupported,
			"unsupported", "Provedor não disponibiliza leitura oficial para esta conta")}
	}

	if record.Type == TypeAntigravity {
		return []ProviderQuotaSnapshot{newPendingSnapshot(record, ScopeAccount, antigravityQuotaSourceName, nowStr, refreshing)}
	}
	if record.Type == TypeCursor {
		return []ProviderQuotaSnapshot{newPendingSnapshot(record, ScopeAccount, cursorUsageQuotaSourceName, nowStr, refreshing)}
	}
	if record.Type == TypeOpenCodeGo {
		return []ProviderQuotaSnapshot{newPendingSnapshot(record, ScopeSubscription, openCodeQuotaSourceName, nowStr, refreshing)}
	}

	// Default / OpenRouter
	snaps := []ProviderQuotaSnapshot{newPendingSnapshot(record, ScopeAPIKey, "openrouter_key_api", nowStr, refreshing)}
	if _, ok := adapter.(AccountQuotaAdapter); ok {
		snaps = append(snaps, newPendingSnapshot(record, ScopeAccount, "openrouter_credits_api", nowStr, refreshing))
	}
	return snaps
}

func newPendingSnapshot(record Record, scope DataScope, sourceName, nowStr string, refreshing bool) ProviderQuotaSnapshot {
	return ProviderQuotaSnapshot{
		AccountID:    record.ID,
		AccountLabel: record.Name,
		ProviderID:   record.Type,
		ProviderName: record.Name,
		Status:       QuotaStatusError,
		Scope:        scope,
		Quotas:       []ProviderQuotaMetric{},
		Source: ProviderQuotaSource{
			Type:      "provider_api",
			Name:      sourceName,
			Scope:     scope,
			FetchedAt: nowStr,
		},
		Error: &ProviderQuotaError{
			Code:    "pending_initial_fetch",
			Message: "Coleta inicial de franquia em andamento",
		},
		Refreshing: refreshing,
	}
}

func isSnapshotSetDisplayable(snapshots []ProviderQuotaSnapshot) bool {
	for _, snap := range snapshots {
		if snap.Status == QuotaStatusAvailable && len(snap.Quotas) > 0 {
			return true
		}
	}
	return false
}
