package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/metering"
)

// Store persists provider usage rows in PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New builds a PostgreSQL metering store.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Insert writes one row idempotently by execution_id.
func (s *Store) Insert(ctx context.Context, record metering.Record) (bool, error) {
	var cost any
	if record.ProviderReportedCost.Valid {
		cost = record.ProviderReportedCost
	}
	tag, err := s.pool.Exec(ctx, `
INSERT INTO provider_execution_usage (
    execution_id, provider_id, model_id,
    input_tokens, output_tokens, total_tokens, cached_input_tokens,
    provider_reported_cost, currency, usage_status, usage_source, recorded_at
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7,
    $8, NULLIF($9, ''), $10, $11, $12
)
ON CONFLICT (execution_id) DO NOTHING`,
		record.ExecutionID,
		record.ProviderID,
		record.ModelID,
		record.InputTokens,
		record.OutputTokens,
		record.TotalTokens,
		record.CachedInputTokens,
		cost,
		record.Currency,
		string(record.Status),
		string(record.Source),
		record.RecordedAt.UTC(),
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// Get returns one persisted row.
func (s *Store) Get(ctx context.Context, executionID string) (metering.Record, error) {
	var record metering.Record
	var status, source string
	var cost pgtype.Numeric
	err := s.pool.QueryRow(ctx, `
SELECT execution_id, provider_id, model_id,
       input_tokens, output_tokens, total_tokens, cached_input_tokens,
       provider_reported_cost, COALESCE(currency, ''), usage_status, usage_source, recorded_at
  FROM provider_execution_usage
 WHERE execution_id = $1`, executionID).Scan(
		&record.ExecutionID,
		&record.ProviderID,
		&record.ModelID,
		&record.InputTokens,
		&record.OutputTokens,
		&record.TotalTokens,
		&record.CachedInputTokens,
		&cost,
		&record.Currency,
		&status,
		&source,
		&record.RecordedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return metering.Record{}, metering.ErrNotFound
	}
	if err != nil {
		return metering.Record{}, err
	}
	record.ProviderReportedCost = cost
	record.Status = metering.Status(status)
	record.Source = metering.Source(source)
	return record, nil
}
