package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"aihub/internal/db"
	"aihub/internal/db/dbtest"
)

// The 0002 rollback must leave a database that the runner can migrate forward
// again: the version row is removed together with the management columns.
func TestMigration0002RollbackAndReapplyIntegration(t *testing.T) {
	pool := dbtest.Pool(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	down4, err := os.ReadFile("migrations/0004_concurrency.down.sql")
	if err != nil {
		t.Fatalf("read down migration 0004: %v", err)
	}
	if _, err := pool.Exec(ctx, string(down4)); err != nil {
		t.Fatalf("apply down migration 0004: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM schema_migrations WHERE version = 4`); err != nil {
		t.Fatalf("forget migration 0004: %v", err)
	}

	down3, err := os.ReadFile("migrations/0003_resource_admission.down.sql")
	if err != nil {
		t.Fatalf("read down migration 0003: %v", err)
	}
	if _, err := pool.Exec(ctx, string(down3)); err != nil {
		t.Fatalf("apply down migration 0003: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM schema_migrations WHERE version = 3`); err != nil {
		t.Fatalf("forget migration 0003: %v", err)
	}

	down, err := os.ReadFile("migrations/0002_provider_management.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}

	var columns int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'providers' AND column_name IN ('name', 'type', 'endpoint', 'enabled', 'config', 'credential_ciphertext')`,
	).Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("management columns after rollback = %d, err %v; want 0", columns, err)
	}

	var highest int
	if err := pool.QueryRow(ctx, `SELECT max(version) FROM schema_migrations`).Scan(&highest); err != nil || highest != 1 {
		t.Fatalf("schema version after rollback = %d, err %v; want 1", highest, err)
	}

	applied, version, err := db.Migrate(ctx, pool)
	if err != nil {
		t.Fatalf("re-apply Migrate: %v", err)
	}

	if len(applied) != 3 || applied[0] != 2 || applied[1] != 3 || applied[2] != 4 || version != 4 {
		t.Fatalf("re-apply = %v version %d, want [2 3 4] version 4", applied, version)
	}
}
