package pgstore

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/execution"
	"aihub/internal/generations"
)

func TestStoreRoundTripsNullableValuesAndCleansOrphansIntegration(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	store := New(pool)
	retainedID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	orphanID := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	streaming := false
	zero := int64(0)
	eventTime := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	retained := generations.TelemetryInput{
		ExecutionID: retainedID,
		ProviderID:  "opencode-go",
		ModelID:     "vendor/model",
		SessionID:   "session-test",
		Result: execution.Result{
			Streaming:         &streaming,
			ProviderLatencyNS: &zero,
			TelemetryEvents: []execution.TelemetryEvent{
				{Kind: execution.EventRoutingSelected, ObservedAt: eventTime, ProviderID: "opencode-go", ModelID: "vendor/model"},
				{Kind: execution.EventGenerationFinished, ObservedAt: eventTime.Add(time.Second), ProviderID: "opencode-go", ModelID: "vendor/model", DurationNS: &zero},
			},
		},
	}
	if err := store.Insert(ctx, retained); err != nil {
		t.Fatalf("insert retained telemetry: %v", err)
	}
	orphan := retained
	orphan.ExecutionID = orphanID
	if err := store.Insert(ctx, orphan); err != nil {
		t.Fatalf("insert orphan telemetry: %v", err)
	}
	for _, executionID := range []string{retainedID, orphanID} {
		if _, err := pool.Exec(ctx, `
INSERT INTO provider_execution_usage (
    execution_id, provider_id, model_id, input_tokens, output_tokens,
    provider_reported_cost, currency, usage_status, usage_source
) VALUES ($1, 'opencode-go', 'vendor/model', 0, NULL, 0, 'USD', 'partial', 'provider_test')`, executionID); err != nil {
			t.Fatalf("insert provider usage for %s: %v", executionID, err)
		}
	}

	items, err := store.List(ctx, []string{retainedID})
	if err != nil {
		t.Fatalf("list enrichment: %v", err)
	}
	item := items[retainedID]
	if !item.HasTelemetry || !item.HasUsage {
		t.Fatalf("enrichment presence = telemetry %t, usage %t", item.HasTelemetry, item.HasUsage)
	}
	if item.Telemetry.Streaming == nil || *item.Telemetry.Streaming {
		t.Fatalf("streaming = %v, want explicit false", item.Telemetry.Streaming)
	}
	if item.Telemetry.ProviderLatencyNS == nil || *item.Telemetry.ProviderLatencyNS != 0 {
		t.Fatalf("provider latency = %v, want explicit zero", item.Telemetry.ProviderLatencyNS)
	}
	if item.Telemetry.ProviderSessionID == nil || *item.Telemetry.ProviderSessionID != "session-test" {
		t.Fatalf("provider session ID = %v, want validated session metadata", item.Telemetry.ProviderSessionID)
	}
	if item.Usage.InputTokens == nil || *item.Usage.InputTokens != 0 || item.Usage.OutputTokens != nil {
		t.Fatalf("token values = input %v, output %v, want zero and null", item.Usage.InputTokens, item.Usage.OutputTokens)
	}
	if item.Usage.ProviderReportedCost == nil || item.Usage.Currency == nil || *item.Usage.Currency != "USD" {
		t.Fatalf("provider cost and currency = %v, %v", item.Usage.ProviderReportedCost, item.Usage.Currency)
	}
	if len(item.Events) != 2 || item.Events[0].Sequence != 0 || item.Events[1].Sequence != 1 {
		t.Fatalf("event order = %+v, want sequences [0 1]", item.Events)
	}

	if err := store.DeleteExcept(ctx, []string{retainedID}); err != nil {
		t.Fatalf("delete orphan enrichment: %v", err)
	}
	assertRowCount(t, ctx, pool, "generation_telemetry", retainedID, 1)
	assertRowCount(t, ctx, pool, "generation_telemetry", orphanID, 0)
	assertRowCount(t, ctx, pool, "provider_execution_usage", retainedID, 1)
	assertRowCount(t, ctx, pool, "provider_execution_usage", orphanID, 0)
}

func assertRowCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, executionID string, want int) {
	t.Helper()
	var got int
	query := "SELECT count(*) FROM " + table + " WHERE execution_id = $1"
	if err := pool.QueryRow(ctx, query, executionID).Scan(&got); err != nil {
		t.Fatalf("count %s rows for %s: %v", table, executionID, err)
	}
	if got != want {
		t.Fatalf("%s row count for %s = %d, want %d", table, executionID, got, want)
	}
}
