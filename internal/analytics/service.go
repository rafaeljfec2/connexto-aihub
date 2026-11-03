package analytics

import (
	"context"
	"fmt"
	"strings"
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
