package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// migrations embeds the versioned migration files:
// internal/db/migrations/NNNN_name.up.sql (and .down.sql files for rollback,
// unused at runtime).
//
//go:embed migrations
var migrations embed.FS

// advisoryKey is the session-level advisory lock key serializing concurrent
// migration runners.
const advisoryKey int64 = 0x6169487562 // "aihub"

// schemaMigrationsTable is created before applying anything so that a fresh
// database records progress and a rolled-back database restarts clean.
const createSchemaMigrations = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    integer PRIMARY KEY CHECK (version > 0),
    name       text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
)`

// Execer is the slice of the transaction creator this runner needs.
type Execer interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// migration is one parsed versioned file.
type migration struct {
	version int
	name    string
	sql     string
}

// Migrate brings the schema up to the latest embedded version. Every file is
// applied in its own transaction and recorded in schema_migrations; a session
// advisory lock serializes concurrent runners. It returns the versions applied
// by this call and the resulting schema version (0 on a fresh database).
func Migrate(ctx context.Context, pool *pgxpool.Pool) (applied []int, schemaVersion int, err error) {
	files, err := loadMigrations()
	if err != nil {
		return nil, 0, err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, 0, errf("acquire connection: %w", err)
	}

	defer conn.Release()

	direct := conn.Conn()

	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if _, unlockErr := direct.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", advisoryKey); unlockErr != nil {
			err = errors.Join(err, errf("release advisory lock: %w", unlockErr))
		}
	}()

	if _, err := direct.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryKey); err != nil {
		return nil, 0, errf("acquire advisory lock: %w", err)
	}

	if _, err := direct.Exec(ctx, createSchemaMigrations); err != nil {
		return nil, 0, errf("create schema bookkeeping: %w", err)
	}

	state, err := currentVersion(ctx, direct)
	if err != nil {
		return nil, 0, err
	}

	if err := validateAppliedSet(state, files); err != nil {
		return nil, state.lastApplied, err
	}

	for _, file := range files {
		if containsInt(state.appliedVersions, file.version) {
			continue
		}

		if err := applyMigration(ctx, direct, file); err != nil {
			return state.appliedVersions, state.lastApplied, err
		}

		state.appliedVersions = append(state.appliedVersions, file.version)
		state.lastApplied = file.version
		applied = append(applied, file.version)
	}

	sort.Ints(state.appliedVersions)

	return applied, state.lastApplied, nil
}

func applyMigration(ctx context.Context, direct Execer, file migration) error {
	tx, err := direct.Begin(ctx)
	if err != nil {
		return errf("begin migration %d: %w", file.version, err)
	}

	if _, err := tx.Exec(ctx, file.sql); err != nil {
		_ = tx.Rollback(ctx)

		return fmt.Errorf("apply migration %d (%s): %w", file.version, file.name, err)
	}

	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", file.version, file.name); err != nil {
		_ = tx.Rollback(ctx)

		return fmt.Errorf("record migration %d: %w", file.version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %d: %w", file.version, err)
	}

	return nil
}

// validateAppliedSet refuses a database whose bookkeeping disagrees with the
// embedded chain: unknown recorded versions or hidden gaps (ADR-020).
func validateAppliedSet(state versionState, files []migration) error {
	versions := make(map[int]struct{}, len(files))
	for _, file := range files {
		versions[file.version] = struct{}{}
	}

	highest := 0
	for _, version := range state.appliedVersions {
		if _, embedded := versions[version]; !embedded {
			return errf("schema_migrations records version %d that has no embedded migration file", version)
		}

		if version > highest {
			highest = version
		}
	}

	for version := 1; version <= highest; version++ {
		if !containsInt(state.appliedVersions, version) {
			return errf("schema_migrations is missing version %d below the highest applied version %d",
				version, highest)
		}
	}

	return nil
}

type versionState struct {
	appliedVersions []int
	lastApplied     int
}

func currentVersion(ctx context.Context, direct Execer) (versionState, error) {
	rows, err := direct.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return versionState{}, errf("read schema bookkeeping: %w", err)
	}

	defer rows.Close()

	state := versionState{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return versionState{}, errf("scan schema bookkeeping: %w", err)
		}

		state.appliedVersions = append(state.appliedVersions, version)
	}

	if err := rows.Err(); err != nil {
		return versionState{}, errf("read schema bookkeeping: %w", err)
	}

	sort.Ints(state.appliedVersions)

	if len(state.appliedVersions) > 0 {
		state.lastApplied = state.appliedVersions[len(state.appliedVersions)-1]
	}

	return state, nil
}

// loadMigrations reads and parses every embedded *.up.sql file. Naming is
// NNNN_name.up.sql and versions must be contiguous from 1 (hidden gaps are
// rejected by this check too).
func loadMigrations() ([]migration, error) {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return nil, errf("list embedded migrations: %w", err)
	}

	var found []migration

	for _, entry := range entries {
		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ".up.sql") {
			continue
		}

		parsed, err := parseMigrationName(fileName)
		if err != nil {
			return nil, err
		}

		content, err := migrations.ReadFile("migrations/" + fileName)
		if err != nil {
			return nil, errf("read migration %s: %w", fileName, err)
		}

		parsed.sql = string(content)
		found = append(found, parsed)
	}

	sort.Slice(found, func(i, j int) bool { return found[i].version < found[j].version })

	previous := 0
	for index := range found {
		if found[index].version != previous+1 {
			return nil, fmt.Errorf("embedded migrations have a version gap: %d then %d",
				previous, found[index].version)
		}

		previous = found[index].version
	}

	return found, nil
}

func parseMigrationName(fileName string) (migration, error) {
	prefix := strings.TrimSuffix(fileName, ".up.sql")
	separator := strings.Index(prefix, "_")
	if separator <= 0 {
		return migration{}, fmt.Errorf("migration file name %s is not NNNN_name.up.sql", fileName)
	}

	version, err := strconv.Atoi(prefix[:separator])
	if err != nil || version <= 0 {
		return migration{}, fmt.Errorf("migration file name %s has a non-positive version", fileName)
	}

	return migration{
		version: version,
		name:    prefix[separator+1:],
	}, nil
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
