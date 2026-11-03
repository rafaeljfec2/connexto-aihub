package analytics

import "context"

// Repository reads admission aggregates from PostgreSQL.
type Repository interface {
	CountAdmitted(ctx context.Context, query RangeQuery) (int, error)
	ListDailyAdmissions(ctx context.Context, query RangeQuery) ([]DailyAdmissionCount, error)
	ListProviderDaily(ctx context.Context, query RangeQuery) ([]ProviderDailyCount, error)
	ListModelDaily(ctx context.Context, query RangeQuery) ([]ModelDailyCount, error)
	ProviderUsageSummary(ctx context.Context, query RangeQuery) (ProviderUsageSummary, error)
	ProviderUsageDaily(ctx context.Context, query RangeQuery) ([]ProviderUsageDailyTokens, error)
	ProviderUsageByProvider(ctx context.Context, query RangeQuery) ([]ProviderUsageProviderDaily, error)
	ProviderUsageByModel(ctx context.Context, query RangeQuery) ([]ProviderUsageModelDaily, error)
}
