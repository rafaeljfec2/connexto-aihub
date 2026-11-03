package analytics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"aihub/internal/analytics"
)

type stubRepo struct {
	count    int
	daily    []analytics.DailyAdmissionCount
	provider []analytics.ProviderDailyCount
	model    []analytics.ModelDailyCount
}

func (s *stubRepo) CountAdmitted(context.Context, analytics.RangeQuery) (int, error) {
	return s.count, nil
}
func (s *stubRepo) ListDailyAdmissions(context.Context, analytics.RangeQuery) ([]analytics.DailyAdmissionCount, error) {
	return s.daily, nil
}
func (s *stubRepo) ListProviderDaily(context.Context, analytics.RangeQuery) ([]analytics.ProviderDailyCount, error) {
	return s.provider, nil
}
func (s *stubRepo) ListModelDaily(context.Context, analytics.RangeQuery) ([]analytics.ModelDailyCount, error) {
	return s.model, nil
}
func (s *stubRepo) ProviderUsageSummary(context.Context, analytics.RangeQuery) (analytics.ProviderUsageSummary, error) {
	return analytics.ProviderUsageSummary{}, nil
}
func (s *stubRepo) ProviderUsageDaily(context.Context, analytics.RangeQuery) ([]analytics.ProviderUsageDailyTokens, error) {
	return nil, nil
}
func (s *stubRepo) ProviderUsageByProvider(context.Context, analytics.RangeQuery) ([]analytics.ProviderUsageProviderDaily, error) {
	return nil, nil
}
func (s *stubRepo) ProviderUsageByModel(context.Context, analytics.RangeQuery) ([]analytics.ProviderUsageModelDaily, error) {
	return nil, nil
}

func TestFillDailySeriesInsertsZeroDays(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	sparse := []analytics.DailyAdmissionCount{
		{PeriodStart: start, AdmittedCount: 2},
		{PeriodStart: end, AdmittedCount: 1},
	}
	series := analytics.FillDailySeries(start, end, sparse)
	if len(series) != 3 {
		t.Fatalf("len = %d, want 3", len(series))
	}
	if series[1].AdmittedCount != 0 {
		t.Fatalf("middle day = %d, want 0", series[1].AdmittedCount)
	}
}

func TestAdmissionsGapFillsAndTotals(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	service := analytics.NewService(&stubRepo{
		count: 3,
		daily: []analytics.DailyAdmissionCount{{PeriodStart: start, AdmittedCount: 3}},
	})
	report, err := service.Admissions(context.Background(), analytics.RangeQuery{From: start, To: end})
	if err != nil {
		t.Fatalf("Admissions: %v", err)
	}
	if report.TotalAdmitted != 3 || len(report.Daily) != 2 || report.Daily[1].AdmittedCount != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestAdmissionsRejectsModelWithoutProvider(t *testing.T) {
	service := analytics.NewService(&stubRepo{})
	_, err := service.Admissions(context.Background(), analytics.RangeQuery{
		From:    time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		To:      time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		ModelID: "m1",
	})
	if err == nil || !errors.Is(err, analytics.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDistributionRejectsModelWithoutProvider(t *testing.T) {
	service := analytics.NewService(&stubRepo{})
	_, err := service.Distribution(context.Background(), analytics.RangeQuery{
		From:    time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		To:      time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		ModelID: "m1",
	})
	if err == nil || !errors.Is(err, analytics.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
