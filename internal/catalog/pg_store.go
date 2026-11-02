package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists the catalog snapshot in PostgreSQL (ADR-020). It
// mirrors the file snapshot contract: Load never returns an error — every
// failure degrades to an empty snapshot with a warning and keeps writes
// enabled (a transient network error must not suspend synchronization) —
// and Save replaces the whole snapshot atomically in one transaction.
type PostgresStore struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPostgresStore builds the PostgreSQL persistence for the catalog. The
// pool must already be open (the database module migrates it first).
func NewPostgresStore(pool *pgxpool.Pool, logger *slog.Logger) *PostgresStore {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &PostgresStore{pool: pool, logger: logger}
}

const (
	postgresSelectProviders = `
SELECT id, last_attempt_at, last_success_at, outcome, error_class, upstream_status,
       discovered_count, added_count, updated_count, unchanged_count, missing_count
FROM providers ORDER BY id`

	postgresSelectModels = `
SELECT provider_id, model_id, canonical_id, name, type, family, modalities,
       limits_context, limits_input, limits_output, capabilities,
       provider_metadata, first_seen_at, last_seen_at, missing_since
FROM models ORDER BY provider_id, model_id`

	postgresSelectSavedAt = `SELECT saved_at FROM catalog_metadata WHERE id = 1`
)

// Load reads the persisted snapshot. A missing state is an empty snapshot;
// a storage failure degrades to empty without disabling writes; an
// invalid state is validated out into an empty snapshot so the next accepted
// sync rewrites it (a documented divergence from file-store quarantine).
func (s *PostgresStore) Load(ctx context.Context) (Snapshot, LoadResult) {
	empty := newEmptySnapshot()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logLoadFailure(err)

		return empty, LoadResult{}
	}

	defer func() { _ = tx.Rollback(ctx) }()

	snapshot, result, loadErr := s.readSnapshot(ctx, tx)
	if loadErr != nil {
		s.logLoadFailure(loadErr)

		return empty, LoadResult{}
	}

	if err := snapshot.validate(); err != nil {
		s.logger.Warn("catalog persisted state failed validation; starting empty",
			"error", err)

		return empty, LoadResult{}
	}

	return snapshot, result
}

func (s *PostgresStore) readSnapshot(ctx context.Context, tx pgx.Tx) (Snapshot, LoadResult, error) {
	providers, err := s.readProviderSnapshot(ctx, tx)
	if err != nil {
		return Snapshot{}, LoadResult{}, err
	}

	models, err := s.readModelRows(ctx, tx)
	if err != nil {
		return Snapshot{}, LoadResult{}, err
	}

	for providerID, entries := range models {
		provider, present := providers[providerID]
		if !present {
			return Snapshot{}, LoadResult{}, errf("model rows reference unknown provider %q", providerID)
		}

		provider.Models = entries
		providers[providerID] = provider
	}

	snapshot := Snapshot{
		Version:   SnapshotVersion,
		Providers: providers,
	}

	if err := tx.QueryRow(ctx, postgresSelectSavedAt).Scan(&snapshot.SavedAt); err != nil && !errIsNoRows(err) {
		return Snapshot{}, LoadResult{}, err
	}

	snapshot.SavedAt = snapshot.SavedAt.UTC()

	return snapshot, LoadResult{}, nil
}

// Save replaces the entire persisted snapshot atomically. FK cascade from
// providers to models keeps the two DELETE/INSERT phases consistent. The
// caller context decides the deadline (the manager passes sync_timeout).
func (s *PostgresStore) Save(ctx context.Context, snapshot Snapshot) (SaveResult, error) {
	if err := ctx.Err(); err != nil {
		return SaveResult{}, err
	}

	snapshot.SavedAt = orCurrentTime(snapshot.SavedAt)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SaveResult{}, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "DELETE FROM providers"); err != nil {
		return SaveResult{}, errf("clear providers: %w", err)
	}

	if err := s.writeProviders(ctx, tx, snapshot.Providers); err != nil {
		return SaveResult{}, err
	}

	if err := s.writeModels(ctx, tx, snapshot.Providers); err != nil {
		return SaveResult{}, err
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO catalog_metadata (id, saved_at) VALUES (1, $1) ON CONFLICT (id) DO UPDATE SET saved_at = EXCLUDED.saved_at",
		snapshot.SavedAt); err != nil {
		return SaveResult{}, errf("record saved_at: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return SaveResult{}, errf("commit catalog snapshot: %w", err)
	}

	return SaveResult{}, nil
}

func (s *PostgresStore) logLoadFailure(loadErr error) {
	s.logger.Warn("catalog persisted state unavailable; starting empty",
		"error", loadErr.Error())
}

func errIsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func orCurrentTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value
}

// rawJSON preserves keys but normalizes formatting like the JSON snapshot
// file does on its own write/read cycle.
func rawJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, errf("encode json value: %w", err)
	}

	return data, nil
}
