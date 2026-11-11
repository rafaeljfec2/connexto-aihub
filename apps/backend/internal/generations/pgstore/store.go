package pgstore

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/generations"
)

// Store persists bounded generation timing and provider event metadata.
type Store struct {
	pool *pgxpool.Pool
}

// New builds a PostgreSQL generation telemetry store.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Insert atomically replaces one execution's telemetry and ordered events.
func (s *Store) Insert(ctx context.Context, input generations.TelemetryInput) error {
	if err := generations.ValidateTelemetry(input); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
INSERT INTO generation_telemetry (
    execution_id, provider_session_id, streaming,
    provider_latency_ns, routing_overhead_ns, time_to_first_token_ns,
    output_stream_duration_ns, finish_reason
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (execution_id) DO UPDATE SET
    provider_session_id = EXCLUDED.provider_session_id,
    streaming = EXCLUDED.streaming,
    provider_latency_ns = EXCLUDED.provider_latency_ns,
    routing_overhead_ns = EXCLUDED.routing_overhead_ns,
    time_to_first_token_ns = EXCLUDED.time_to_first_token_ns,
    output_stream_duration_ns = EXCLUDED.output_stream_duration_ns,
    finish_reason = EXCLUDED.finish_reason,
    recorded_at = now()`,
		input.ExecutionID, generations.SafeSessionID(input.SessionID), input.Result.Streaming,
		input.Result.ProviderLatencyNS, input.Result.RoutingOverheadNS, input.Result.TimeToFirstTokenNS,
		input.Result.OutputStreamDurationNS, input.FinishReason,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM generation_provider_events WHERE execution_id = $1`, input.ExecutionID); err != nil {
		return err
	}
	for sequence, event := range input.Result.TelemetryEvents {
		if _, err := tx.Exec(ctx, `
INSERT INTO generation_provider_events (
    execution_id, sequence, event_type, observed_at,
    provider_id, model_id, upstream_status, duration_ns
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			input.ExecutionID, sequence, string(event.Kind), event.ObservedAt.UTC(),
			event.ProviderID, event.ModelID, event.UpstreamStatus, event.DurationNS,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DeleteExcept removes telemetry whose E06 execution record is no longer retained.
func (s *Store) DeleteExcept(ctx context.Context, retainedIDs []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM generation_telemetry WHERE NOT (execution_id = ANY($1::text[]))`, retainedIDs); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM provider_execution_usage WHERE NOT (execution_id = ANY($1::text[]))`, retainedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
