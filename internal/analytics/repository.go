package analytics

import "context"

// Repository reads admission aggregates from PostgreSQL.
type Repository interface {
	CountAdmitted(ctx context.Context, query RangeQuery) (int, error)
	ListDailyAdmissions(ctx context.Context, query RangeQuery) ([]DailyAdmissionCount, error)
	ListProviderDaily(ctx context.Context, query RangeQuery) ([]ProviderDailyCount, error)
	ListModelDaily(ctx context.Context, query RangeQuery) ([]ModelDailyCount, error)
}
