// Package db_test hosts the database integration tests. It is an external
// test package because it uses internal/db/dbtest, which depends on
// internal/db itself.
package db_test

import (
	"context"
	"testing"
	"time"

	"aihub/internal/db"
	"aihub/internal/db/dbtest"
)

// The integration tests below require a real PostgreSQL server per ADR-020
// and opt in through AIHUB_TEST_DATABASE_URL; dbtest gives every test its own
// throwaway database with the embedded migrations applied.

func TestMigrationsApplyOnceAndAreIdempotentIntegration(t *testing.T) {
	pool := dbtest.Pool(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	firstApplied, firstVersion, err := db.Migrate(ctx, pool)
	if err != nil {
		t.Fatalf("first Migrate: %v", err)
	}

	if firstVersion != 9 {
		t.Fatalf("schema_version after first run = %d, want 9", firstVersion)
	}

	if len(firstApplied) != 9 || firstApplied[0] != 1 || firstApplied[1] != 2 || firstApplied[2] != 3 || firstApplied[3] != 4 || firstApplied[4] != 5 || firstApplied[5] != 6 || firstApplied[6] != 7 || firstApplied[7] != 8 || firstApplied[8] != 9 {
		t.Fatalf("applied versions = %v, want exactly [1 2 3 4 5 6 7 8 9]", firstApplied)
	}

	secondApplied, secondVersion, err := db.Migrate(ctx, pool)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	if secondVersion != 9 {
		t.Fatalf("schema_version after second run = %d, want 9", secondVersion)
	}

	if len(secondApplied) != 0 {
		t.Fatalf("second run re-applied %v, want an empty list", secondApplied)
	}
}

func TestMigrateRejectsForeignBookkeepingVersionsIntegration(t *testing.T) {
	pool := dbtest.Pool(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if _, _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	if _, err := pool.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES (99, 'foreign')"); err != nil {
		t.Fatalf("insert foreign version: %v", err)
	}

	applied, version, err := db.Migrate(ctx, pool)
	if err == nil {
		t.Fatalf("Migrate with foreign bookkeeping version must fail, got version %d applied %v", version, applied)
	}
}
