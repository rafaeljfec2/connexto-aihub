package pgstore

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/analytics"
)

// Store reads analytics aggregates from PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New builds a PostgreSQL analytics store.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CountAdmitted(ctx context.Context, query analytics.RangeQuery) (int, error) {
	sql, args := admissionFilter(`
SELECT count(*)::int
  FROM resource_admissions
 WHERE allowed = true`, query)
	var count int
	if err := s.pool.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) ListDailyAdmissions(ctx context.Context, query analytics.RangeQuery) ([]analytics.DailyAdmissionCount, error) {
	sql, args := admissionFilter(`
SELECT period_start, count(*)::int
  FROM resource_admissions
 WHERE allowed = true`, query)
	sql += `
 GROUP BY period_start
 ORDER BY period_start`

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []analytics.DailyAdmissionCount
	for rows.Next() {
		var day time.Time
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		out = append(out, analytics.DailyAdmissionCount{
			PeriodStart: utcDay(day), AdmittedCount: count,
		})
	}
	return out, rows.Err()
}

func (s *Store) ListProviderDaily(ctx context.Context, query analytics.RangeQuery) ([]analytics.ProviderDailyCount, error) {
	sql, args := admissionFilter(`
SELECT period_start, provider_id, count(*)::int
  FROM resource_admissions
 WHERE allowed = true`, query)
	sql += `
 GROUP BY period_start, provider_id
 ORDER BY period_start, provider_id`

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []analytics.ProviderDailyCount
	for rows.Next() {
		var day time.Time
		var providerID string
		var count int
		if err := rows.Scan(&day, &providerID, &count); err != nil {
			return nil, err
		}
		out = append(out, analytics.ProviderDailyCount{
			PeriodStart: utcDay(day), ProviderID: providerID, AdmittedCount: count,
		})
	}
	return out, rows.Err()
}

func (s *Store) ListModelDaily(ctx context.Context, query analytics.RangeQuery) ([]analytics.ModelDailyCount, error) {
	sql, args := admissionFilter(`
SELECT period_start, provider_id, model_id, count(*)::int
  FROM resource_admissions
 WHERE allowed = true`, query)
	sql += `
 GROUP BY period_start, provider_id, model_id
 ORDER BY period_start, provider_id, model_id`

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []analytics.ModelDailyCount
	for rows.Next() {
		var day time.Time
		var providerID, modelID string
		var count int
		if err := rows.Scan(&day, &providerID, &modelID, &count); err != nil {
			return nil, err
		}
		out = append(out, analytics.ModelDailyCount{
			PeriodStart: utcDay(day), ProviderID: providerID, ModelID: modelID, AdmittedCount: count,
		})
	}
	return out, rows.Err()
}

func admissionFilter(base string, query analytics.RangeQuery) (string, []any) {
	args := []any{query.From, query.To}
	filter := base + `
   AND period_start >= $1
   AND period_start <= $2`
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

func utcDay(day time.Time) time.Time {
	utc := day.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
