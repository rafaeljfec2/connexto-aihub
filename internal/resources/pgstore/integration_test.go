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

	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/providers"
	"aihub/internal/resources"
	"aihub/internal/resources/pgstore"
)

func migratedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

func TestMigration0003AppliesIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	var tables int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables
		WHERE table_name IN ('resource_policies', 'resource_period_counters', 'resource_admissions')`).Scan(&tables); err != nil || tables != 3 {
		t.Fatalf("resource tables = %d, err %v; want 3", tables, err)
	}
}

func TestPolicyCRUDAndScopeUniquenessIntegration(t *testing.T) {
	pool := migratedPool(t)
	store := pgstore.New(pool)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(store, resources.WithClock(func() time.Time { return now }))

	global, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 100, Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy global: %v", err)
	}
	if _, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 50, Enabled: true,
	}); err == nil || !errors.Is(err, resources.ErrConflict) {
		t.Fatalf("expected scope conflict, got %v", err)
	}

	provider, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope:      resources.ResourceScope{Kind: resources.ScopeProvider, ProviderID: providers.OpenCodeGoID},
		DailyLimit: 10,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy provider: %v", err)
	}
	if _, err := service.UpdatePolicy(ctx, provider.ID, resources.UpdatePolicyInput{DailyLimit: intPtr(5)}); err != nil {
		t.Fatalf("UpdatePolicy: %v", err)
	}
	if _, err := service.SetPolicyEnabled(ctx, global.ID, false); err != nil {
		t.Fatalf("SetPolicyEnabled: %v", err)
	}
	list, err := service.ListPolicies(ctx)
	if err != nil || len(list) < 2 {
		t.Fatalf("ListPolicies = %d policies, err %v", len(list), err)
	}
}

func TestAdmissionIncrementsAllApplicableScopesIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))

	mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 10, Enabled: true,
	})
	mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope:      resources.ResourceScope{Kind: resources.ScopeProvider, ProviderID: providers.OpenCodeGoID},
		DailyLimit: 10,
		Enabled:    true,
	})
	mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeModel, ProviderID: providers.OpenCodeGoID, ModelID: "m1"},
		DailyLimit: 10, Enabled: true,
	})

	decision, err := service.Admit(ctx, resources.AdmissionRequest{ID: "adm-a", ProviderID: providers.OpenCodeGoID, ModelID: "m1"})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if decision.GlobalCount != 1 || decision.ProviderCount != 1 || decision.ModelCount != 1 {
		t.Fatalf("counts = %+v", decision)
	}
}

func TestAdmissionRejectsWithoutIncrementIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 1, Enabled: true,
	})
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: "adm-1", ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("first Admit: %v", err)
	}
	_, err := service.Admit(ctx, resources.AdmissionRequest{ID: "adm-2", ProviderID: providers.OpenCodeGoID, ModelID: "m1"})
	if !resources.IsLimitExceeded(err) {
		t.Fatalf("second Admit error = %v, want limit exceeded", err)
	}
	var count int
	period := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	if err := pool.QueryRow(ctx, `SELECT admitted_count FROM resource_period_counters c
		JOIN resource_policies p ON p.id = c.policy_id
		WHERE p.scope = 'global' AND c.period_start = $1`, period).Scan(&count); err != nil || count != 1 {
		t.Fatalf("global counter = %d, err %v; want 1 after rejection", count, err)
	}
}

func TestAdmissionIdempotencyIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 5, Enabled: true,
	})
	req := resources.AdmissionRequest{ID: "adm-repeat", ProviderID: providers.OpenCodeGoID, ModelID: "m1"}
	if _, err := service.Admit(ctx, req); err != nil {
		t.Fatalf("first Admit: %v", err)
	}
	if _, err := service.Admit(ctx, req); err != nil {
		t.Fatalf("retry Admit: %v", err)
	}
	var count int
	period := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	if err := pool.QueryRow(ctx, `SELECT admitted_count FROM resource_period_counters c
		JOIN resource_policies p ON p.id = c.policy_id WHERE p.scope = 'global' AND c.period_start = $1`, period).Scan(&count); err != nil || count != 1 {
		t.Fatalf("counter after idempotent retry = %d, err %v", count, err)
	}
}

func TestConcurrentLastUnitAdmissionIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 1, Enabled: true,
	})

	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			id, err := resources.NewAdmissionID()
			if err != nil {
				return
			}
			_, err = service.Admit(ctx, resources.AdmissionRequest{
				ID: id, ProviderID: providers.OpenCodeGoID, ModelID: fmt.Sprintf("model-%d", index),
			})
			if err == nil {
				allowed.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if allowed.Load() != 1 {
		t.Fatalf("allowed admissions = %d, want exactly 1", allowed.Load())
	}
}

func TestDisabledPolicyAndMissingPolicyDoNotBlockIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	policy, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 1, Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if _, err := service.SetPolicyEnabled(ctx, policy.ID, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	for i := 0; i < 3; i++ {
		id, _ := resources.NewAdmissionID()
		if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
			t.Fatalf("Admit %d: %v", i, err)
		}
	}
}

func TestPolicyChangeAffectsFutureAdmissionsOnlyIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	policy := mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 2, Enabled: true,
	})
	id1, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id1, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("first Admit: %v", err)
	}
	if _, err := service.UpdatePolicy(ctx, policy.ID, resources.UpdatePolicyInput{DailyLimit: intPtr(1)}); err != nil {
		t.Fatalf("tighten limit: %v", err)
	}
	id2, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id2, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); !resources.IsLimitExceeded(err) {
		t.Fatalf("second Admit after tighten = %v, want limit exceeded", err)
	}
}

func TestProviderModelPoliciesAreIsolatedByProviderIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	sharedModel := "shared-model-id"
	mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope:      resources.ResourceScope{Kind: resources.ScopeModel, ProviderID: providers.OpenCodeGoID, ModelID: sharedModel},
		DailyLimit: 1,
		Enabled:    true,
	})
	mustCreatePolicy(t, service, ctx, resources.NewPolicyInput{
		Scope:      resources.ResourceScope{Kind: resources.ScopeModel, ProviderID: providers.OpenRouterID, ModelID: sharedModel},
		DailyLimit: 1,
		Enabled:    true,
	})

	id1, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id1, ProviderID: providers.OpenCodeGoID, ModelID: sharedModel}); err != nil {
		t.Fatalf("OpenCode Admit: %v", err)
	}
	id2, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id2, ProviderID: providers.OpenCodeGoID, ModelID: sharedModel}); !resources.IsLimitExceeded(err) {
		t.Fatalf("second OpenCode Admit = %v, want limit exceeded", err)
	}
	id3, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id3, ProviderID: providers.OpenRouterID, ModelID: sharedModel}); err != nil {
		t.Fatalf("OpenRouter Admit with same model id = %v, want allowed", err)
	}
}

func TestAdmissionRollsOverAtUTCPeriodBoundaryIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	endOfDay := time.Date(2026, 11, 3, 23, 59, 59, 0, time.UTC)
	startNextDay := time.Date(2026, 11, 4, 0, 0, 1, 0, time.UTC)
	dayOne := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return endOfDay }))
	mustCreatePolicy(t, dayOne, ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 1, Enabled: true,
	})
	id1, _ := resources.NewAdmissionID()
	if _, err := dayOne.Admit(ctx, resources.AdmissionRequest{ID: id1, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("first day Admit: %v", err)
	}
	id2, _ := resources.NewAdmissionID()
	if _, err := dayOne.Admit(ctx, resources.AdmissionRequest{ID: id2, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); !resources.IsLimitExceeded(err) {
		t.Fatalf("second same-day Admit = %v, want limit exceeded", err)
	}

	dayTwo := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return startNextDay }))
	id3, _ := resources.NewAdmissionID()
	if _, err := dayTwo.Admit(ctx, resources.AdmissionRequest{ID: id3, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("next UTC day Admit = %v, want allowed", err)
	}
}

func TestAdmissionSurvivesServiceRestartIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	first := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	mustCreatePolicy(t, first, ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 2, Enabled: true,
	})
	id, _ := resources.NewAdmissionID()
	if _, err := first.Admit(ctx, resources.AdmissionRequest{ID: id, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	restarted := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	if _, err := restarted.Admit(ctx, resources.AdmissionRequest{ID: id, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("idempotent restart Admit: %v", err)
	}
	id2, _ := resources.NewAdmissionID()
	if _, err := restarted.Admit(ctx, resources.AdmissionRequest{ID: id2, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("second fresh Admit: %v", err)
	}
	id3, _ := resources.NewAdmissionID()
	if _, err := restarted.Admit(ctx, resources.AdmissionRequest{ID: id3, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); !resources.IsLimitExceeded(err) {
		t.Fatalf("third Admit = %v, want limit exceeded", err)
	}
}

func TestUpsertPolicyByScopeAndUsageIntegration(t *testing.T) {
	pool := migratedPool(t)
	store := pgstore.New(pool)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	service := resources.NewService(store, resources.WithClock(func() time.Time { return now }))

	policy, err := service.UpsertPolicy(ctx, resources.UpsertPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 3, Enabled: true,
	})
	if err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}
	found, err := service.GetPolicyByScope(ctx, resources.ResourceScope{Kind: resources.ScopeGlobal})
	if err != nil || found.ID != policy.ID {
		t.Fatalf("GetPolicyByScope = %+v err %v", found, err)
	}

	id, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("Admit: %v", err)
	}

	usage, err := service.ListUsage(ctx, resources.UsageQuery{})
	if err != nil || len(usage) != 1 || usage[0].AdmittedCount != 1 {
		t.Fatalf("usage = %+v err %v", usage, err)
	}
}

func TestUpsertPolicyUpdatePreservesCountersIntegration(t *testing.T) {
	pool := migratedPool(t)
	store := pgstore.New(pool)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 16, 0, 0, 0, time.UTC)
	service := resources.NewService(store, resources.WithClock(func() time.Time { return now }))

	if _, err := service.UpsertPolicy(ctx, resources.UpsertPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 5, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}
	id, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if _, err := service.UpsertPolicy(ctx, resources.UpsertPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 10, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy update: %v", err)
	}
	usage, err := service.ListUsage(ctx, resources.UsageQuery{})
	if err != nil || len(usage) != 1 || usage[0].AdmittedCount != 1 {
		t.Fatalf("usage after limit update = %+v err %v", usage, err)
	}
}

func mustCreatePolicy(t *testing.T, service *resources.Service, ctx context.Context, input resources.NewPolicyInput) resources.ResourcePolicy {
	t.Helper()
	policy, err := service.CreatePolicy(ctx, input)
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	return policy
}

func intPtr(value int) *int { return &value }
