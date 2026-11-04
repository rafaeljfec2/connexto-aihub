package pgstore

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"aihub/internal/generations"
)

// List returns enrichment for a bounded set of execution IDs using two batch queries.
func (s *Store) List(ctx context.Context, executionIDs []string) (map[string]generations.Enrichment, error) {
	result := make(map[string]generations.Enrichment, len(executionIDs))
	if len(executionIDs) == 0 {
		return result, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT requested.execution_id,
       t.execution_id IS NOT NULL,
       u.execution_id IS NOT NULL,
       t.provider_session_id, t.streaming,
       t.provider_latency_ns, t.routing_overhead_ns,
       t.time_to_first_token_ns, t.output_stream_duration_ns, t.finish_reason,
       u.input_tokens, u.output_tokens, u.provider_reported_cost, u.provider_list_cost,
       u.currency, u.usage_status, u.usage_source
  FROM unnest($1::text[]) AS requested(execution_id)
  LEFT JOIN generation_telemetry t ON t.execution_id = requested.execution_id
  LEFT JOIN provider_execution_usage u ON u.execution_id = requested.execution_id`, executionIDs)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var executionID string
		var hasTelemetry, hasUsage bool
		var sessionID, finishReason, currency, usageStatus, usageSource pgtype.Text
		var streaming pgtype.Bool
		var providerLatency, routingOverhead, firstToken, outputDuration pgtype.Int8
		var inputTokens, outputTokens pgtype.Int8
		var reportedCost, listCost pgtype.Numeric
		if err := rows.Scan(&executionID, &hasTelemetry, &hasUsage, &sessionID, &streaming,
			&providerLatency, &routingOverhead, &firstToken, &outputDuration, &finishReason,
			&inputTokens, &outputTokens, &reportedCost, &listCost, &currency, &usageStatus, &usageSource); err != nil {
			rows.Close()
			return nil, err
		}
		item := result[executionID]
		item.HasTelemetry = hasTelemetry
		item.HasUsage = hasUsage
		if hasTelemetry {
			item.Telemetry = generations.Telemetry{
				Streaming:              boolValue(streaming),
				ProviderLatencyNS:      int64Value(providerLatency),
				RoutingOverheadNS:      int64Value(routingOverhead),
				TimeToFirstTokenNS:     int64Value(firstToken),
				OutputStreamDurationNS: int64Value(outputDuration),
				FinishReason:           textValue(finishReason),
				ProviderSessionID:      textValue(sessionID),
			}
		}
		if hasUsage {
			item.Usage = generations.Usage{
				Status:               usageStatus.String,
				Source:               usageSource.String,
				InputTokens:          int64Value(inputTokens),
				OutputTokens:         int64Value(outputTokens),
				ProviderReportedCost: numericValue(reportedCost),
				ProviderListCost:     numericValue(listCost),
				Currency:             textValue(currency),
			}
		}
		result[executionID] = item
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if err := s.loadEvents(ctx, executionIDs, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) loadEvents(ctx context.Context, executionIDs []string, result map[string]generations.Enrichment) error {
	rows, err := s.pool.Query(ctx, `
SELECT execution_id, sequence, event_type, observed_at,
       provider_id, model_id, upstream_status, duration_ns
  FROM generation_provider_events
 WHERE execution_id = ANY($1::text[])
 ORDER BY execution_id, sequence`, executionIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var executionID, eventType, providerID, modelID string
		var sequence int16
		var observedAt pgtype.Timestamptz
		var upstreamStatus pgtype.Int4
		var duration pgtype.Int8
		if err := rows.Scan(&executionID, &sequence, &eventType, &observedAt,
			&providerID, &modelID, &upstreamStatus, &duration); err != nil {
			return err
		}
		item := result[executionID]
		item.Events = append(item.Events, generations.ProviderEvent{
			Sequence:       int(sequence),
			Type:           eventType,
			ObservedAt:     observedAt.Time.UTC(),
			ProviderID:     providerID,
			ModelID:        modelID,
			UpstreamStatus: int32Value(upstreamStatus),
			DurationNS:     int64Value(duration),
		})
		result[executionID] = item
	}
	return rows.Err()
}

func int64Value(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func int32Value(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int32)
	return &converted
}

func textValue(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func boolValue(value pgtype.Bool) *bool {
	if !value.Valid {
		return nil
	}
	return &value.Bool
}

func numericValue(value pgtype.Numeric) *string {
	if !value.Valid {
		return nil
	}
	encoded, err := value.MarshalJSON()
	if err != nil {
		return nil
	}
	formatted := string(encoded)
	return &formatted
}
