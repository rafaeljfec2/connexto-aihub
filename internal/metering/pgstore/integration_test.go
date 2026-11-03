package pgstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/metering"
	meteringpg "aihub/internal/metering/pgstore"
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

func TestInsertIsIdempotentByExecutionID(t *testing.T) {
	pool := migratedPool(t)
	store := meteringpg.New(pool)
	ctx := context.Background()
	record := metering.Record{
		ExecutionID: "exec-1",
		ProviderID:  providers.OpenCodeGoID,
		ModelID:     "model-a",
		InputTokens: int64Ptr(3),
		OutputTokens: int64Ptr(2),
		TotalTokens: int64Ptr(5),
		Status:      metering.StatusReported,
		Source:      metering.SourceOpenCodeGo,
		RecordedAt:  time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC),
	}
	inserted, err := store.Insert(ctx, record)
	if err != nil || !inserted {
		t.Fatalf("first insert = %v err %v", inserted, err)
	}
	inserted, err = store.Insert(ctx, record)
	if err != nil || inserted {
		t.Fatalf("second insert = %v err %v", inserted, err)
	}
	got, err := store.Get(ctx, "exec-1")
	if err != nil || got.InputTokens == nil || *got.InputTokens != 3 {
		t.Fatalf("got = %+v err %v", got, err)
	}
}

func TestInsertUnavailableWithoutTokenColumns(t *testing.T) {
	pool := migratedPool(t)
	store := meteringpg.New(pool)
	ctx := context.Background()
	_, err := store.Insert(ctx, metering.Record{
		ExecutionID: "exec-2",
		ProviderID:  providers.OpenCodeGoID,
		ModelID:     "model-a",
		Status:      metering.StatusUnavailable,
		Source:      metering.SourceOpenCodeGo,
		RecordedAt:  time.Date(2026, 11, 3, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := store.Get(ctx, "exec-2")
	if err != nil || got.Status != metering.StatusUnavailable || got.InputTokens != nil {
		t.Fatalf("got = %+v err %v", got, err)
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestInsertStoresReportedCostAsNumeric(t *testing.T) {
	pool := migratedPool(t)
	store := meteringpg.New(pool)
	var cost pgtype.Numeric
	if err := cost.Scan("0.0000168"); err != nil {
		t.Fatal(err)
	}
	_, err := store.Insert(context.Background(), metering.Record{
		ExecutionID:          "exec-cost",
		ProviderID:           providers.OpenCodeGoID,
		ModelID:              "model-a",
		InputTokens:          int64Ptr(1),
		OutputTokens:         int64Ptr(1),
		TotalTokens:          int64Ptr(2),
		ProviderReportedCost: cost,
		Currency:             "USD",
		Status:               metering.StatusReported,
		Source:               metering.SourceOpenCodeGo,
		RecordedAt:           time.Date(2026, 11, 3, 14, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
}
