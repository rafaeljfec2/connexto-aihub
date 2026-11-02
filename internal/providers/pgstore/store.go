// Package pgstore persists provider configurations in PostgreSQL (Phase C1).
// It lives outside the providers domain so the domain never imports pgx.
package pgstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/providers"
)

// uniqueViolation is the SQLSTATE of a unique constraint failure.
const uniqueViolation = "23505"

// Store implements providers.Repository over the providers table. Rows that
// only carry catalog sync state (no management name) are invisible here.
type Store struct {
	pool *pgxpool.Pool
}

// New builds the store on an open pool whose schema is already migrated.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const (
	managedColumns = `id, name, type, endpoint, enabled, credential_ciphertext IS NOT NULL, created_at, updated_at`

	insertManaged = `
INSERT INTO providers (id, name, type, endpoint, enabled, credential_ciphertext,
                       outcome, error_class, upstream_status)
VALUES ($1, $2, $3, $4, $5, $6, 'never', 'none', 0)
RETURNING ` + managedColumns

	ensureManaged = `
INSERT INTO providers (id, name, type, endpoint, enabled, credential_ciphertext,
                       outcome, error_class, upstream_status)
VALUES ($1, $2, $3, $4, $5, $6, 'never', 'none', 0)
ON CONFLICT (id) DO UPDATE SET
    name                 = COALESCE(providers.name, EXCLUDED.name),
    type                 = COALESCE(providers.type, EXCLUDED.type),
    endpoint             = COALESCE(providers.endpoint, EXCLUDED.endpoint),
    credential_ciphertext = COALESCE(providers.credential_ciphertext, EXCLUDED.credential_ciphertext),
    enabled              = CASE WHEN providers.name IS NULL THEN EXCLUDED.enabled ELSE providers.enabled END,
    updated_at           = CASE WHEN providers.name IS NULL THEN now() ELSE providers.updated_at END
RETURNING ` + managedColumns

	selectManaged = `SELECT ` + managedColumns + ` FROM providers WHERE id = $1 AND name IS NOT NULL`

	selectAllManaged = `SELECT ` + managedColumns + ` FROM providers WHERE name IS NOT NULL ORDER BY id`

	updateManaged = `
UPDATE providers SET
    name                  = COALESCE($2::text, name),
    endpoint              = COALESCE($3::text, endpoint),
    credential_ciphertext = COALESCE($4::bytea, credential_ciphertext),
    updated_at            = now()
WHERE id = $1 AND name IS NOT NULL
RETURNING ` + managedColumns

	setEnabled = `
UPDATE providers SET enabled = $2, updated_at = now()
WHERE id = $1 AND name IS NOT NULL
RETURNING ` + managedColumns

	selectSealed = `SELECT credential_ciphertext FROM providers WHERE id = $1 AND name IS NOT NULL`
)

// Create inserts a new configuration.
func (s *Store) Create(ctx context.Context, record providers.NewRecord) (providers.Record, error) {
	row := s.pool.QueryRow(ctx, insertManaged, record.ID, record.Name, record.Type,
		record.Endpoint, record.Enabled, record.Credential)

	return mapWriteError(scanRecord(row))
}

// EnsureConfigured seeds a configuration without overwriting persisted values.
func (s *Store) EnsureConfigured(ctx context.Context, record providers.NewRecord) (providers.Record, error) {
	row := s.pool.QueryRow(ctx, ensureManaged, record.ID, record.Name, record.Type,
		record.Endpoint, record.Enabled, record.Credential)

	return mapWriteError(scanRecord(row))
}

// Get returns one configuration.
func (s *Store) Get(ctx context.Context, id string) (providers.Record, error) {
	return scanRecord(s.pool.QueryRow(ctx, selectManaged, id))
}

// List returns every configuration ordered by id.
func (s *Store) List(ctx context.Context) ([]providers.Record, error) {
	rows, err := s.pool.Query(ctx, selectAllManaged)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	defer rows.Close()

	records := make([]providers.Record, 0)

	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	return records, nil
}

// Update applies name, endpoint and credential changes in one statement; nil
// fields keep their stored value.
func (s *Store) Update(ctx context.Context, id string, changes providers.Changes) (providers.Record, error) {
	row := s.pool.QueryRow(ctx, updateManaged, id, changes.Name, changes.Endpoint, changes.Credential)

	return mapWriteError(scanRecord(row))
}

// SetEnabled changes the enabled flag.
func (s *Store) SetEnabled(ctx context.Context, id string, enabled bool) (providers.Record, error) {
	return scanRecord(s.pool.QueryRow(ctx, setEnabled, id, enabled))
}

// SealedCredential returns the stored ciphertext, or nil when none exists.
func (s *Store) SealedCredential(ctx context.Context, id string) ([]byte, error) {
	var sealed []byte

	if err := s.pool.QueryRow(ctx, selectSealed, id).Scan(&sealed); err != nil {
		return nil, translateNotFound(err)
	}

	return sealed, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(row rowScanner) (providers.Record, error) {
	var record providers.Record

	if err := row.Scan(&record.ID, &record.Name, &record.Type, &record.Endpoint,
		&record.Enabled, &record.HasCredential, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return providers.Record{}, translateNotFound(err)
	}

	return record, nil
}

func translateNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return providers.ErrNotFound
	}

	return fmt.Errorf("query provider: %w", err)
}

func mapWriteError(record providers.Record, err error) (providers.Record, error) {
	if err == nil {
		return record, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return providers.Record{}, providers.ErrConflict
	}

	return providers.Record{}, err
}
