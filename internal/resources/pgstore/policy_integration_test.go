package pgstore_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"aihub/internal/providers"
	"aihub/internal/resources"
	"aihub/internal/resources/pgstore"
)

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

func TestUpsertPolicyByScopeAndUsageIntegration(t *testing.T) {
	pool := migratedPool(t)
	store := pgstore.New(pool)
	ctx := context.Background()
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	service := resources.NewService(store, resources.WithClock(func() time.Time { return now }))

	policy, _, err := service.UpsertPolicy(ctx, resources.UpsertPolicyInput{
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

	if _, _, err := service.UpsertPolicy(ctx, resources.UpsertPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 5, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}
	id, _ := resources.NewAdmissionID()
	if _, err := service.Admit(ctx, resources.AdmissionRequest{ID: id, ProviderID: providers.OpenCodeGoID, ModelID: "m1"}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if _, _, err := service.UpsertPolicy(ctx, resources.UpsertPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 10, Enabled: true,
	}); err != nil {
		t.Fatalf("UpsertPolicy update: %v", err)
	}
	usage, err := service.ListUsage(ctx, resources.UsageQuery{})
	if err != nil || len(usage) != 1 || usage[0].AdmittedCount != 1 {
		t.Fatalf("usage after limit update = %+v err %v", usage, err)
	}
}

func intPtr(value int) *int { return &value }
