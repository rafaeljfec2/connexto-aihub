package pgstore_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/concurrency"
	"aihub/internal/concurrency/pgstore"
	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/providers"
)

func migratedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

func TestConcurrentAcquireLastSlotIntegration(t *testing.T) {
	pool := migratedPool(t)
	store := pgstore.New(pool)
	service := concurrency.NewService(store, concurrency.WithLeaseTTL(time.Minute))
	ctx := context.Background()

	if _, _, err := service.UpsertPolicy(ctx, concurrency.UpsertPolicyInput{
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("exec-%d", n)
			_, err := service.Acquire(ctx, concurrency.AcquireRequest{
				ExecutionID: id, ProviderID: providers.OpenCodeGoID, ModelID: "m1",
			})
			if err == nil {
				allowed.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if allowed.Load() != 1 {
		t.Fatalf("allowed = %d, want 1", allowed.Load())
	}
}

func TestAcquireReleaseIdempotentIntegration(t *testing.T) {
	pool := migratedPool(t)
	store := pgstore.New(pool)
	service := concurrency.NewService(store)
	ctx := context.Background()

	req := concurrency.AcquireRequest{ExecutionID: "exec-a", ProviderID: providers.OpenCodeGoID, ModelID: "m1"}
	if _, err := service.Acquire(ctx, req); err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if _, err := service.Acquire(ctx, req); err != nil {
		t.Fatalf("idempotent Acquire: %v", err)
	}
	if err := service.Release(ctx, "exec-a"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := service.Release(ctx, "exec-a"); err != nil {
		t.Fatalf("idempotent Release: %v", err)
	}
}

func TestExpiredLeasePurgedOnNextAcquireIntegration(t *testing.T) {
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
		Scope: concurrency.Scope{Kind: concurrency.ScopeGlobal}, ConcurrencyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	if _, err := service.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "stale-exec", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	clock = start.Add(2 * time.Minute)
	if _, err := service.Acquire(ctx, concurrency.AcquireRequest{
		ExecutionID: "fresh-exec", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("second Acquire after expiry: %v", err)
	}
}
