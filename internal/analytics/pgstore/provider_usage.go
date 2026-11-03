package pgstore

import (
	"context"
	"fmt"
	"time"

	"aihub/internal/analytics"
)

func (s *Store) ProviderUsageSummary(ctx context.Context, query analytics.RangeQuery) (analytics.ProviderUsageSummary, error) {
	sql, args := providerUsageFilter(`
SELECT
       SUM(input_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       SUM(output_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       SUM(total_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       count(*) FILTER (WHERE usage_status = 'reported')::int,
       count(*) FILTER (WHERE usage_status = 'unavailable')::int,
       count(*) FILTER (WHERE usage_status = 'partial')::int,
       count(*) FILTER (WHERE usage_status = 'invalid')::int
  FROM provider_execution_usage
 WHERE recorded_at >= $1
   AND recorded_at < $2`, query)
	var summary analytics.ProviderUsageSummary
	summary.From = query.From
	summary.To = query.To
	err := s.pool.QueryRow(ctx, sql, args...).Scan(
		&summary.InputTokens,
		&summary.OutputTokens,
		&summary.TotalTokens,
		&summary.ReportedExecutions,
		&summary.UnavailableExecutions,
		&summary.PartialExecutions,
		&summary.InvalidExecutions,
	)
	return summary, err
}

func (s *Store) ProviderUsageDaily(ctx context.Context, query analytics.RangeQuery) ([]analytics.ProviderUsageDailyTokens, error) {
	sql, args := providerUsageFilter(`
SELECT date_trunc('day', recorded_at AT TIME ZONE 'UTC')::date,
       SUM(input_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       SUM(output_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       SUM(total_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       count(*)::int
  FROM provider_execution_usage
 WHERE recorded_at >= $1
   AND recorded_at < $2`, query)
	sql += `
 GROUP BY 1
 ORDER BY 1`

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []analytics.ProviderUsageDailyTokens
	for rows.Next() {
		var day time.Time
		var row analytics.ProviderUsageDailyTokens
		if err := rows.Scan(&day, &row.InputTokens, &row.OutputTokens, &row.TotalTokens, &row.ExecutionCount); err != nil {
			return nil, err
		}
		row.PeriodStart = utcDay(day)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) ProviderUsageByProvider(ctx context.Context, query analytics.RangeQuery) ([]analytics.ProviderUsageProviderDaily, error) {
	sql, args := providerUsageFilter(`
SELECT date_trunc('day', recorded_at AT TIME ZONE 'UTC')::date,
       provider_id,
       SUM(input_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       SUM(output_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       SUM(total_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       count(*)::int
  FROM provider_execution_usage
 WHERE recorded_at >= $1
   AND recorded_at < $2`, query)
	sql += `
 GROUP BY 1, provider_id
 ORDER BY 1, provider_id`

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []analytics.ProviderUsageProviderDaily
	for rows.Next() {
		var day time.Time
		var row analytics.ProviderUsageProviderDaily
		if err := rows.Scan(&day, &row.ProviderID, &row.InputTokens, &row.OutputTokens, &row.TotalTokens, &row.Executions); err != nil {
			return nil, err
		}
		row.PeriodStart = utcDay(day)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) ProviderUsageByModel(ctx context.Context, query analytics.RangeQuery) ([]analytics.ProviderUsageModelDaily, error) {
	sql, args := providerUsageFilter(`
SELECT date_trunc('day', recorded_at AT TIME ZONE 'UTC')::date,
       provider_id,
       model_id,
       SUM(input_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       SUM(output_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       SUM(total_tokens) FILTER (WHERE usage_status IN ('reported', 'partial'))::bigint,
       count(*)::int
  FROM provider_execution_usage
 WHERE recorded_at >= $1
   AND recorded_at < $2`, query)
	sql += `
 GROUP BY 1, provider_id, model_id
 ORDER BY 1, provider_id, model_id`

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []analytics.ProviderUsageModelDaily
	for rows.Next() {
		var day time.Time
		var row analytics.ProviderUsageModelDaily
		if err := rows.Scan(&day, &row.ProviderID, &row.ModelID, &row.InputTokens, &row.OutputTokens, &row.TotalTokens, &row.Executions); err != nil {
			return nil, err
		}
		row.PeriodStart = utcDay(day)
		out = append(out, row)
	}
	return out, rows.Err()
}

func providerUsageFilter(base string, query analytics.RangeQuery) (string, []any) {
	start := query.From.UTC()
	endExclusive := query.To.UTC().Add(24 * time.Hour)
	args := []any{start, endExclusive}
	filter := base
	if query.ProviderID != "" {
		args = append(args, query.ProviderID)
		filter += fmt.Sprintf(`
   AND provider_id = $%d`, len(args))
	}
	if query.ModelID != "" {
		args = append(args, query.ModelID)
		filter += fmt.Sprintf(`
   AND model_id = $%d`, len(args))
	}
	return filter, args
}
