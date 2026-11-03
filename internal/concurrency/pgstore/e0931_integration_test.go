package pgstore_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/concurrency"
	"aihub/internal/concurrency/pgstore"
	"aihub/internal/providers"
)

func secondPoolSameDatabase(t *testing.T, first *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	second, err := pgxpool.NewWithConfig(context.Background(), first.Config())
	if err != nil {
		t.Fatalf("second pool: %v", err)
	}
	t.Cleanup(second.Close)
	return second
}

func activeLeaseCountAt(t *testing.T, pool *pgxpool.Pool, now time.Time) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*)::int FROM concurrency_leases WHERE expires_at > $1`, now.UTC()).Scan(&count); err != nil {
		t.Fatalf("active leases: %v", err)
	}
	return count
}

func TestE0931ScenarioCTwoIndependentPoolsIntegration(t *testing.T) {
	poolA := migratedPool(t)
	poolB := secondPoolSameDatabase(t, poolA)
	storeA := pgstore.New(poolA)
	storeB := pgstore.New(poolB)
	serviceA := concurrency.NewService(storeA)
	serviceB := concurrency.NewService(storeB)
	ctx := context.Background()

	if _, _, err := serviceA.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i, svc := range []*concurrency.Service{serviceA, serviceB} {
		wg.Add(1)
		go func(n int, service *concurrency.Service) {
			defer wg.Done()
			_, err := service.Acquire(ctx, concurrency.AcquireRequest{
				ExecutionID: fmt.Sprintf("inst-%d", n),
				ProviderID:  providers.OpenCodeGoID,
				ModelID:     "m1",
			})
			if err == nil {
				accepted.Add(1)
			}
		}(i, svc)
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("accepted = %d, want 1", accepted.Load())
	}
	if activeLeaseCountAt(t, poolA, time.Now().UTC()) != 1 {
		t.Fatalf("active leases after race != 1")
	}

	var holder string
	if err := poolA.QueryRow(ctx, `SELECT execution_id FROM concurrency_leases LIMIT 1`).Scan(&holder); err != nil {
		t.Fatalf("lookup holder: %v", err)
	}
	if err := serviceB.Release(ctx, holder); err != nil {
		t.Fatalf("release holder from instance B: %v", err)
	}
	if activeLeaseCountAt(t, poolA, time.Now().UTC()) != 0 {
		t.Fatalf("leases after release want 0")
	}
	if _, err := serviceA.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "after-release", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("acquire after release from instance A: %v", err)
	}
}

func TestE0931ScenarioDLeaseLifecycleIntegration(t *testing.T) {
	pool := migratedPool(t)
	store := pgstore.New(pool)
	start := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	clock := start
	ttl := time.Minute
	service := concurrency.NewService(store,
		concurrency.WithClock(func() time.Time { return clock }),
		concurrency.WithLeaseTTL(ttl),
	)
	ctx := context.Background()

	if _, _, err := service.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	if _, err := service.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "exec-lease", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if activeLeaseCountAt(t, pool, clock) != 1 {
		t.Fatalf("want one active lease after acquire")
	}

	clock = start.Add(30 * time.Second)
	if err := service.RenewLease(ctx, "exec-lease"); err != nil {
		t.Fatalf("RenewLease: %v", err)
	}
	clock = start.Add(89 * time.Second)
	if activeLeaseCountAt(t, pool, clock) != 1 {
		t.Fatalf("renewed lease should remain active before extended expiry")
	}

	if err := service.RenewLease(ctx, "missing-exec"); !errors.Is(err, concurrency.ErrNotFound) {
		t.Fatalf("renew missing = %v, want ErrNotFound", err)
	}

	if err := service.Release(ctx, "exec-lease"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := service.Release(ctx, "exec-lease"); err != nil {
		t.Fatalf("idempotent Release: %v", err)
	}
	if activeLeaseCountAt(t, pool, clock) != 0 {
		t.Fatalf("leases after release = %d, want 0", activeLeaseCountAt(t, pool, clock))
	}

	// Abandoned lease: expires via purge on next acquire.
	if _, err := service.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "abandoned", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("Acquire abandoned: %v", err)
	}
	clock = start.Add(3 * time.Minute)
	if _, err := service.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "after-purge", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("Acquire after expiry purge: %v", err)
	}
	if activeLeaseCountAt(t, pool, clock) != 1 {
		t.Fatalf("want one lease after purge replaced abandoned slot")
	}
}

func TestE0931ScenarioDRenewedLeaseNotPurgedWithStalePeerIntegration(t *testing.T) {
	pool := migratedPool(t)
	store := pgstore.New(pool)
	start := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	clock := start
	service := concurrency.NewService(store,
		concurrency.WithClock(func() time.Time { return clock }),
		concurrency.WithLeaseTTL(time.Minute),
	)
	ctx := context.Background()

	if _, _, err := service.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 2, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	if _, err := service.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "stale", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("Acquire stale: %v", err)
	}
	if _, err := service.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "active", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("Acquire active: %v", err)
	}

	clock = start.Add(50 * time.Second)
	if err := service.RenewLease(ctx, "active"); err != nil {
		t.Fatalf("Renew active: %v", err)
	}
	clock = start.Add(105 * time.Second)
	if _, err := service.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "third", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("third acquire: %v", err)
	}
	if activeLeaseCountAt(t, pool, clock) != 2 {
		t.Fatalf("active leases = %d, want stale purged and active+third remaining", activeLeaseCountAt(t, pool, clock))
	}
}
