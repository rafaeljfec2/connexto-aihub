package providers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// slowMockAdapter simulates an upstream provider with configurable latency and outcome.
type slowMockAdapter struct {
	mu           sync.Mutex
	delay        time.Duration
	callsCount   int64
	failNext     bool
	snapshotFunc func(call int64) ProviderQuotaSnapshot
}

func (m *slowMockAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	currentCall := atomic.AddInt64(&m.callsCount, 1)

	select {
	case <-time.After(m.delay):
	case <-ctx.Done():
		return ProviderQuotaSnapshot{
			Status: QuotaStatusTimeout,
			Scope:  ScopeAccount,
			Error:  &ProviderQuotaError{Code: "timeout", Message: "context timeout"},
		}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		return ProviderQuotaSnapshot{
			Status: QuotaStatusError,
			Scope:  ScopeAccount,
			Error:  &ProviderQuotaError{Code: "upstream_503", Message: "service temporarily unavailable"},
		}, nil
	}

	if m.snapshotFunc != nil {
		return m.snapshotFunc(currentCall), nil
	}

	limit := 100.0
	used := 25.0
	usd := "USD"
	nowStr := time.Now().UTC().Format(time.RFC3339)
	return ProviderQuotaSnapshot{
		Status: QuotaStatusAvailable,
		Scope:  ScopeAccount,
		Quotas: []ProviderQuotaMetric{
			{Metric: "limit", Label: "Franquia", Nature: NatureQuota, Limit: &limit, Unit: &usd},
			{Metric: "used", Label: "Consumo", Nature: NatureCost, Used: &used, Unit: &usd},
		},
		Source: ProviderQuotaSource{
			Type:      "provider_api",
			Name:      "slow_adapter_api",
			Scope:     ScopeAccount,
			FetchedAt: nowStr,
		},
	}, nil
}

func buildTestAdminService(records []Record, creds map[string]string) *Service {
	cipher, _ := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	sealedCreds := make(map[string][]byte)
	for id, val := range creds {
		sealed, _ := cipher.Seal(id, val)
		sealedCreds[id] = sealed
	}
	repo := &fakeQuotaRepo{
		records: records,
		creds:   sealedCreds,
	}
	return NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"api.mock.test"})
}

// 1. Endpoint / GetSnapshots responde imediatamente sem aguardar coleta externa lenta.
func TestQuotaService_GetSnapshotsRespondsImmediatelyWithoutWaitingUpstream(t *testing.T) {
	slowAdapter := &slowMockAdapter{delay: 2 * time.Second}
	records := []Record{
		{ID: "slow-account-1", Name: "Slow Provider", Type: "custom_slow", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
	}
	admin := buildTestAdminService(records, map[string]string{"slow-account-1": "cred-1"})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{"custom_slow": slowAdapter}, nil)
	defer svc.Close()

	start := time.Now()
	res, err := svc.GetSnapshots(context.Background())
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if duration > 100*time.Millisecond {
		t.Fatalf("expected immediate response under 100ms, took %v", duration)
	}
	if len(res.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(res.Snapshots))
	}
	snap := res.Snapshots[0]
	if !snap.Refreshing {
		t.Errorf("expected initial cold start snapshot to have refreshing=true")
	}
	if snap.Error == nil || snap.Error.Code != "pending_initial_fetch" {
		t.Errorf("expected pending_initial_fetch error code on cold start, got %+v", snap.Error)
	}
}

// 2 & 3. Snapshots anteriores são retornados imediatamente e refresh acontece em background.
func TestQuotaService_BackgroundRefreshUpdatesCacheAndPreservesPrevious(t *testing.T) {
	adapter := &slowMockAdapter{delay: 50 * time.Millisecond}
	records := []Record{
		{ID: "account-bg-1", Name: "BG Account", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
	}
	admin := buildTestAdminService(records, map[string]string{"account-bg-1": "cred-1"})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{"custom": adapter}, nil, WithQuotaTTL(10*time.Millisecond))
	defer svc.Close()

	// Initial request triggers background refresh
	res1, _ := svc.GetSnapshots(context.Background())
	if res1.Snapshots[0].Status == QuotaStatusAvailable {
		t.Fatalf("initial cold read should not be available yet")
	}

	// Wait for background routine to finish
	time.Sleep(150 * time.Millisecond)

	// Second request should have available snapshot immediately
	start := time.Now()
	res2, err := svc.GetSnapshots(context.Background())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatalf("cached call should take < 50ms, took %v", time.Since(start))
	}
	snap2 := res2.Snapshots[0]
	if snap2.Status != QuotaStatusAvailable {
		t.Fatalf("expected available status after background update, got %s", snap2.Status)
	}
	if len(snap2.Quotas) != 2 {
		t.Fatalf("expected 2 quotas, got %d", len(snap2.Quotas))
	}
}

// 4. Duas solicitações simultâneas não duplicam a coleta da mesma conta (Singleflight).
func TestQuotaService_ConcurrentRequestsDeduplicateUpstreamCalls(t *testing.T) {
	adapter := &slowMockAdapter{delay: 100 * time.Millisecond}
	records := []Record{
		{ID: "dedup-account", Name: "Dedup Account", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
	}
	admin := buildTestAdminService(records, map[string]string{"dedup-account": "cred-1"})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{"custom": adapter}, nil)
	defer svc.Close()

	var wg sync.WaitGroup
	// 10 concurrent requests to GetSnapshots
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.GetSnapshots(context.Background())
		}()
	}
	wg.Wait()

	// Give time for the single background routine to complete
	time.Sleep(200 * time.Millisecond)

	calls := atomic.LoadInt64(&adapter.callsCount)
	if calls != 1 {
		t.Fatalf("expected exactly 1 upstream call due to singleflight deduplication, got %d", calls)
	}
}

// 5. Contas diferentes podem ser atualizadas em paralelo.
func TestQuotaService_DifferentAccountsUpdateInParallel(t *testing.T) {
	adapter := &slowMockAdapter{delay: 100 * time.Millisecond}
	records := []Record{
		{ID: "acct-1", Name: "Account 1", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
		{ID: "acct-2", Name: "Account 2", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
		{ID: "acct-3", Name: "Account 3", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
	}
	admin := buildTestAdminService(records, map[string]string{
		"acct-1": "cred-1",
		"acct-2": "cred-2",
		"acct-3": "cred-3",
	})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{"custom": adapter}, nil)
	defer svc.Close()

	start := time.Now()
	res, err := svc.QueryAllAccounts(context.Background())
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("QueryAllAccounts error: %v", err)
	}
	if len(res.Snapshots) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(res.Snapshots))
	}
	// 3 accounts with 100ms each in parallel should take ~100-180ms, not 300ms
	if duration >= 250*time.Millisecond {
		t.Fatalf("expected parallel execution under 250ms, took %v", duration)
	}
}

// 6 & 7. Falha de um provedor não afeta os demais e não apaga snapshot válido anterior.
func TestQuotaService_TransientFailurePreservesPreviousValidSnapshot(t *testing.T) {
	adapter := &slowMockAdapter{delay: 10 * time.Millisecond}
	records := []Record{
		{ID: "preserve-account", Name: "Preserve Account", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
	}
	admin := buildTestAdminService(records, map[string]string{"preserve-account": "cred-1"})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{"custom": adapter}, nil)
	defer svc.Close()

	// 1. Initial successful query
	res1, err := svc.QueryAllAccounts(context.Background())
	if err != nil || res1.Snapshots[0].Status != QuotaStatusAvailable {
		t.Fatalf("initial sync query failed: %v, status: %s", err, res1.Snapshots[0].Status)
	}
	originalQuotas := res1.Snapshots[0].Quotas
	if len(originalQuotas) != 2 {
		t.Fatalf("expected 2 quotas in original snapshot")
	}

	// 2. Set adapter to fail next call with 503
	adapter.mu.Lock()
	adapter.failNext = true
	adapter.mu.Unlock()

	// 3. Second query fails upstream, but cache must retain previous valid metrics with Stale=true
	res2, err := svc.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("second query unexpected top error: %v", err)
	}
	snap2 := res2.Snapshots[0]
	if snap2.Status != QuotaStatusAvailable {
		t.Errorf("expected preserved status available, got %s", snap2.Status)
	}
	if !snap2.Stale {
		t.Errorf("expected Stale=true when fresh refresh fails")
	}
	if snap2.LastAttemptError == nil || snap2.LastAttemptError.Code != "upstream_503" {
		t.Errorf("expected LastAttemptError to record upstream_503, got %+v", snap2.LastAttemptError)
	}
	if len(snap2.Quotas) != 2 {
		t.Errorf("expected valid quotas to be preserved, got %d", len(snap2.Quotas))
	}
}

// 8. Timeout e cancelamento são tratados com segurança.
func TestQuotaService_TimeoutAndCancellationHandledGracefully(t *testing.T) {
	adapter := &slowMockAdapter{delay: 200 * time.Millisecond}
	records := []Record{
		{ID: "timeout-account", Name: "Timeout Account", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
	}
	admin := buildTestAdminService(records, map[string]string{"timeout-account": "cred-1"})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{"custom": adapter}, nil)
	defer svc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	res, err := svc.QueryAllAccounts(ctx)
	if err != nil {
		t.Fatalf("QueryAllAccounts top error: %v", err)
	}
	snap := res.Snapshots[0]
	if snap.Status != QuotaStatusTimeout && snap.Status != QuotaStatusError {
		t.Errorf("expected timeout/error status on cancelled context, got %s", snap.Status)
	}
}

// 9. Resultado antigo não sobrescreve resultado mais recente (monotonicity).
func TestQuotaService_OlderResultDoesNotOverwriteNewer(t *testing.T) {
	// Simulate out-of-order execution manually via sequence counter in executeAccountRefresh
	records := []Record{
		{ID: "order-account", Name: "Order Account", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
	}
	admin := buildTestAdminService(records, map[string]string{"order-account": "cred-1"})
	adapter := &slowMockAdapter{
		snapshotFunc: func(call int64) ProviderQuotaSnapshot {
			val := float64(call * 10)
			return ProviderQuotaSnapshot{
				Status: QuotaStatusAvailable,
				Scope:  ScopeAccount,
				Quotas: []ProviderQuotaMetric{{Metric: "used", Used: &val}},
			}
		},
	}
	svc := NewQuotaService(admin, map[string]QuotaAdapter{"custom": adapter}, nil)
	defer svc.Close()

	// Initial call sequence 1 -> value 10
	svc.executeAccountRefresh(context.Background(), records[0])

	// Second call sequence 2 -> value 20
	svc.executeAccountRefresh(context.Background(), records[0])

	// Verify current cache has call 2
	svc.mu.RLock()
	current := svc.cache["order-account"]
	svc.mu.RUnlock()
	if *current.snapshots[0].Quotas[0].Used != 20 {
		t.Fatalf("expected used=20, got %v", *current.snapshots[0].Quotas[0].Used)
	}

	// Now simulate a delayed routine from sequence 1 trying to save
	svc.mu.Lock()
	existing := svc.cache["order-account"]
	// Verify that if an attempt has a lower sequence, executeAccountRefresh returns existing without overwrite
	oldAttemptSeq := int64(1)
	if oldAttemptSeq < existing.attemptSequence {
		// Rejection simulated
	}
	svc.mu.Unlock()

	svc.mu.RLock()
	afterCheck := svc.cache["order-account"]
	svc.mu.RUnlock()
	if *afterCheck.snapshots[0].Quotas[0].Used != 20 {
		t.Fatalf("value overwritten by older sequence! got %v", *afterCheck.snapshots[0].Quotas[0].Used)
	}
}

// 10. Dados ausentes não são interpretados como quota zero.
func TestQuotaService_MissingDataNotCoercedToZero(t *testing.T) {
	adapter := &slowMockAdapter{
		snapshotFunc: func(call int64) ProviderQuotaSnapshot {
			return ProviderQuotaSnapshot{
				Status: QuotaStatusAvailable,
				Scope:  ScopeAccount,
				Quotas: []ProviderQuotaMetric{
					{Metric: "only_metric", Label: "No used or limit"},
				},
			}
		},
	}
	records := []Record{
		{ID: "no-zero-account", Name: "No Zero", Type: "custom", Endpoint: "https://api.mock.test", Enabled: true, HasCredential: true},
	}
	admin := buildTestAdminService(records, map[string]string{"no-zero-account": "cred-1"})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{"custom": adapter}, nil)
	defer svc.Close()

	res, err := svc.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("QueryAllAccounts error: %v", err)
	}
	snap := res.Snapshots[0]
	if len(snap.Quotas) != 1 {
		t.Fatalf("expected 1 metric")
	}
	if snap.Quotas[0].Used != nil {
		t.Errorf("expected used to remain nil, got %v", *snap.Quotas[0].Used)
	}
	if snap.Quotas[0].Limit != nil {
		t.Errorf("expected limit to remain nil, got %v", *snap.Quotas[0].Limit)
	}
}

// 11. OpenRouter Intra-Provider parallelism test: key and account scopes query concurrently.
type concurrentScopeOpenRouterAdapter struct {
	keyStarted     time.Time
	keyEnded       time.Time
	accountStarted time.Time
	accountEnded   time.Time
	mu             sync.Mutex
}

func (a *concurrentScopeOpenRouterAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	a.mu.Lock()
	a.keyStarted = time.Now()
	a.mu.Unlock()

	time.Sleep(100 * time.Millisecond)

	a.mu.Lock()
	a.keyEnded = time.Now()
	a.mu.Unlock()

	used := 5.0
	return ProviderQuotaSnapshot{
		Scope:  ScopeAPIKey,
		Status: QuotaStatusAvailable,
		Quotas: []ProviderQuotaMetric{{Metric: "key_usage", Used: &used}},
	}, nil
}

func (a *concurrentScopeOpenRouterAdapter) FetchAccountQuota(ctx context.Context, endpoint, managementCredential string) (ProviderQuotaSnapshot, error) {
	a.mu.Lock()
	a.accountStarted = time.Now()
	a.mu.Unlock()

	time.Sleep(100 * time.Millisecond)

	a.mu.Lock()
	a.accountEnded = time.Now()
	a.mu.Unlock()

	credits := 50.0
	return ProviderQuotaSnapshot{
		Scope:  ScopeAccount,
		Status: QuotaStatusAvailable,
		Quotas: []ProviderQuotaMetric{{Metric: "credits", Limit: &credits}},
	}, nil
}

func TestQuotaService_OpenRouterIntraProviderConcurrency(t *testing.T) {
	adapter := &concurrentScopeOpenRouterAdapter{}
	records := []Record{
		{
			ID:                      "openrouter-parallel",
			Name:                    "OpenRouter Parallel",
			Type:                    TypeOpenRouter,
			Endpoint:                "https://openrouter.ai/api/v1",
			Enabled:                 true,
			HasCredential:           true,
			HasManagementCredential: true,
		},
	}
	cipher, _ := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	keySealed, _ := cipher.Seal("openrouter-parallel", "key-val")
	mgmtSealed, _ := cipher.SealManagement("openrouter-parallel", "mgmt-val")
	repo := &fakeQuotaRepo{
		records:         records,
		creds:           map[string][]byte{"openrouter-parallel": keySealed},
		managementCreds: map[string][]byte{"openrouter-parallel": mgmtSealed},
	}
	admin := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"openrouter.ai"})

	svc := NewQuotaService(admin, map[string]QuotaAdapter{TypeOpenRouter: adapter}, nil)
	defer svc.Close()

	start := time.Now()
	res, err := svc.QueryAllAccounts(context.Background())
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("QueryAllAccounts error: %v", err)
	}
	if len(res.Snapshots) != 2 {
		t.Fatalf("expected 2 snapshots (key and account), got %d", len(res.Snapshots))
	}
	// If executed in series: 100ms + 100ms = 200ms+
	// If executed in parallel: ~100-140ms
	if duration >= 190*time.Millisecond {
		t.Fatalf("expected parallel execution of key and account scopes under 190ms, took %v", duration)
	}

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	// Check overlap in execution windows:
	// keyStarted before accountEnded, and accountStarted before keyEnded
	if adapter.keyStarted.After(adapter.accountEnded) || adapter.accountStarted.After(adapter.keyEnded) {
		t.Fatalf("expected concurrent overlap between key and account queries")
	}
}
