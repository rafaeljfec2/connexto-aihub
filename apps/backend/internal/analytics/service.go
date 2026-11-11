package analytics

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Service computes analytics reports.
type Service struct {
	repo Repository
}

// NewService builds the analytics service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Admissions returns global admitted volume and a gap-filled daily series.
func (s *Service) Admissions(ctx context.Context, query RangeQuery) (AdmissionsReport, error) {
	start, end, err := NormalizeRangeBounds(query.From, query.To)
	if err != nil {
		return AdmissionsReport{}, err
	}
	query.From = start
	query.To = end
	query.ProviderID = strings.TrimSpace(query.ProviderID)
	query.ModelID = strings.TrimSpace(query.ModelID)
	if query.ModelID != "" && query.ProviderID == "" {
		return AdmissionsReport{}, fmt.Errorf("%w: model_id requires provider_id", ErrInvalidInput)
	}

	total, err := s.repo.CountAdmitted(ctx, query)
	if err != nil {
		return AdmissionsReport{}, err
	}
	sparse, err := s.repo.ListDailyAdmissions(ctx, query)
	if err != nil {
		return AdmissionsReport{}, err
	}
	return AdmissionsReport{
		From: start, To: end, TotalAdmitted: total,
		Daily: FillDailySeries(start, end, sparse),
	}, nil
}

// Distribution returns provider and model daily breakdowns for the range.
func (s *Service) Distribution(ctx context.Context, query RangeQuery) (DistributionReport, error) {
	start, end, err := NormalizeRangeBounds(query.From, query.To)
	if err != nil {
		return DistributionReport{}, err
	}
	query.From = start
	query.To = end
	query.ProviderID = strings.TrimSpace(query.ProviderID)
	query.ModelID = strings.TrimSpace(query.ModelID)
	if query.ModelID != "" && query.ProviderID == "" {
		return DistributionReport{}, fmt.Errorf("%w: model_id requires provider_id", ErrInvalidInput)
	}

	byProvider, err := s.repo.ListProviderDaily(ctx, query)
	if err != nil {
		return DistributionReport{}, err
	}
	byModel, err := s.repo.ListModelDaily(ctx, query)
	if err != nil {
		return DistributionReport{}, err
	}
	return DistributionReport{
		From: start, To: end, ByProvider: byProvider, ByModel: byModel,
	}, nil
}

// ProviderUsage returns token aggregates and status counts for persisted provider usage.
func (s *Service) ProviderUsage(ctx context.Context, query RangeQuery) (ProviderUsageSummary, error) {
	start, end, err := NormalizeRangeBounds(query.From, query.To)
	if err != nil {
		return ProviderUsageSummary{}, err
	}
	query.From = start
	query.To = end
	query.ProviderID = strings.TrimSpace(query.ProviderID)
	query.ModelID = strings.TrimSpace(query.ModelID)
	if query.ModelID != "" && query.ProviderID == "" {
		return ProviderUsageSummary{}, fmt.Errorf("%w: model_id requires provider_id", ErrInvalidInput)
	}
	summary, err := s.repo.ProviderUsageSummary(ctx, query)
	if err != nil {
		return ProviderUsageSummary{}, err
	}
	sparse, err := s.repo.ProviderUsageDaily(ctx, query)
	if err != nil {
		return ProviderUsageSummary{}, err
	}
	summary.From = start
	summary.To = end
	summary.Daily = fillProviderUsageDailySeries(start, end, sparse)
	return summary, nil
}

// ProviderUsageDistribution returns provider/model token breakdowns.
func (s *Service) ProviderUsageDistribution(ctx context.Context, query RangeQuery) (ProviderUsageDistributionReport, error) {
	start, end, err := NormalizeRangeBounds(query.From, query.To)
	if err != nil {
		return ProviderUsageDistributionReport{}, err
	}
	query.From = start
	query.To = end
	query.ProviderID = strings.TrimSpace(query.ProviderID)
	query.ModelID = strings.TrimSpace(query.ModelID)
	if query.ModelID != "" && query.ProviderID == "" {
		return ProviderUsageDistributionReport{}, fmt.Errorf("%w: model_id requires provider_id", ErrInvalidInput)
	}
	byProvider, err := s.repo.ProviderUsageByProvider(ctx, query)
	if err != nil {
		return ProviderUsageDistributionReport{}, err
	}
	byModel, err := s.repo.ProviderUsageByModel(ctx, query)
	if err != nil {
		return ProviderUsageDistributionReport{}, err
	}
	return ProviderUsageDistributionReport{
		From: start, To: end, ByProvider: byProvider, ByModel: byModel,
	}, nil
}

func fillProviderUsageDailySeries(start, end time.Time, sparse []ProviderUsageDailyTokens) []ProviderUsageDailyTokens {
	index := make(map[string]ProviderUsageDailyTokens, len(sparse))
	for _, row := range sparse {
		key := row.PeriodStart.UTC().Format(time.DateOnly)
		index[key] = row
	}
	out := make([]ProviderUsageDailyTokens, 0)
	for day := start; !day.After(end); day = day.Add(24 * time.Hour) {
		key := day.UTC().Format(time.DateOnly)
		if row, ok := index[key]; ok {
			out = append(out, row)
			continue
		}
		out = append(out, ProviderUsageDailyTokens{PeriodStart: day.UTC()})
	}
	return out
}
