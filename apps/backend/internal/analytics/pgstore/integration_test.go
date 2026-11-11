package pgstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/analytics"
	analyticspg "aihub/internal/analytics/pgstore"
	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/metering"
	meteringpg "aihub/internal/metering/pgstore"
	"aihub/internal/providers"
	"aihub/internal/resources"
	"aihub/internal/resources/pgstore"
)

func migratedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

func TestAnalyticsAggregatesAdmittedRowsIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	day := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	resourcesService := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time {
		return day.Add(12 * time.Hour)
	}))
	if _, err := resourcesService.Admit(ctx, resources.AdmissionRequest{
		ID: "adm-1", ProviderID: providers.OpenCodeGoID, ModelID: "m1",
	}); err != nil {
		t.Fatalf("Admit 1: %v", err)
	}
	if _, err := resourcesService.Admit(ctx, resources.AdmissionRequest{
		ID: "adm-2", ProviderID: providers.OpenRouterID, ModelID: "m2",
	}); err != nil {
		t.Fatalf("Admit 2: %v", err)
	}

	analyticsService := analytics.NewService(analyticspg.New(pool))
	report, err := analyticsService.Admissions(ctx, analytics.RangeQuery{From: day, To: day})
	if err != nil {
		t.Fatalf("Admissions: %v", err)
	}
	if report.TotalAdmitted != 2 || report.Daily[0].AdmittedCount != 2 {
		t.Fatalf("report = %+v", report)
	}

	distribution, err := analyticsService.Distribution(ctx, analytics.RangeQuery{From: day, To: day})
	if err != nil {
		t.Fatalf("Distribution: %v", err)
	}
	if len(distribution.ByProvider) != 2 || len(distribution.ByModel) != 2 {
		t.Fatalf("distribution = %+v", distribution)
	}

	providerDistribution, err := analyticsService.Distribution(ctx, analytics.RangeQuery{
		From: day, To: day, ProviderID: providers.OpenCodeGoID,
	})
	if err != nil || len(providerDistribution.ByProvider) != 1 || len(providerDistribution.ByModel) != 1 || providerDistribution.ByModel[0].ModelID != "m1" {
		t.Fatalf("provider-filtered distribution = %+v, err %v", providerDistribution, err)
	}
	modelDistribution, err := analyticsService.Distribution(ctx, analytics.RangeQuery{
		From: day, To: day, ProviderID: providers.OpenRouterID, ModelID: "m2",
	})
	if err != nil || len(modelDistribution.ByProvider) != 1 || len(modelDistribution.ByModel) != 1 || modelDistribution.ByModel[0].ModelID != "m2" {
		t.Fatalf("model-filtered distribution = %+v, err %v", modelDistribution, err)
	}
	emptyDistribution, err := analyticsService.Distribution(ctx, analytics.RangeQuery{
		From: day, To: day, ProviderID: providers.OpenRouterID, ModelID: "missing",
	})
	if err != nil || len(emptyDistribution.ByProvider) != 0 || len(emptyDistribution.ByModel) != 0 {
		t.Fatalf("empty filtered distribution = %+v, err %v", emptyDistribution, err)
	}
}

func TestProviderUsageAggregatesPreserveUnknownAndZeroIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	day := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	store := meteringpg.New(pool)
	zero := int64(0)
	inputTwo, outputThree, totalFive := int64(2), int64(3), int64(5)
	inputFour, inputEight, outputFour := int64(4), int64(8), int64(4)
	records := []metering.Record{
		providerUsageRecord("unknown", "unknown", metering.StatusPartial, nil, nil, nil, day),
		providerUsageRecord("zero", "zero", metering.StatusReported, &zero, &zero, &zero, day),
		providerUsageRecord("mixed-input", "mixed", metering.StatusPartial, &inputEight, nil, nil, day),
		providerUsageRecord("mixed-output", "mixed", metering.StatusPartial, nil, &outputFour, nil, day),
		providerUsageRecord("combined-reported", "combined", metering.StatusReported, &inputTwo, &outputThree, &totalFive, day),
		providerUsageRecord("combined-partial", "combined", metering.StatusPartial, &inputFour, nil, nil, day),
	}
	for _, record := range records {
		if _, err := store.Insert(ctx, record); err != nil {
			t.Fatalf("Insert %q: %v", record.ExecutionID, err)
		}
	}

	service := analytics.NewService(analyticspg.New(pool))
	query := func(modelID string) analytics.ProviderUsageSummary {
		t.Helper()
		report, err := service.ProviderUsage(ctx, analytics.RangeQuery{
			From: day, To: day, ProviderID: providers.OpenCodeGoID, ModelID: modelID,
		})
		if err != nil {
			t.Fatalf("ProviderUsage(%q): %v", modelID, err)
		}
		return report
	}
	assertValue := func(name string, got *int64, want *int64) {
		t.Helper()
		if want == nil {
			if got != nil {
				t.Fatalf("%s = %d, want unknown", name, *got)
			}
			return
		}
		if got == nil || *got != *want {
			t.Fatalf("%s = %v, want %d", name, got, *want)
		}
	}

	unknown := query("unknown")
	assertValue("unknown input", unknown.InputTokens, nil)
	assertValue("unknown output", unknown.OutputTokens, nil)
	assertValue("unknown total", unknown.TotalTokens, nil)
	assertValue("unknown daily input", unknown.Daily[0].InputTokens, nil)

	zeroReport := query("zero")
	assertValue("zero input", zeroReport.InputTokens, &zero)
	assertValue("zero output", zeroReport.OutputTokens, &zero)
	assertValue("zero total", zeroReport.TotalTokens, &zero)

	mixed := query("mixed")
	assertValue("mixed input", mixed.InputTokens, &inputEight)
	assertValue("mixed output", mixed.OutputTokens, &outputFour)
	assertValue("mixed total", mixed.TotalTokens, nil)

	combined := query("combined")
	inputSix := int64(6)
	assertValue("combined input", combined.InputTokens, &inputSix)
	assertValue("combined output", combined.OutputTokens, &outputThree)
	assertValue("combined total", combined.TotalTokens, &totalFive)

	empty := query("missing")
	assertValue("empty input", empty.InputTokens, nil)
	assertValue("empty output", empty.OutputTokens, nil)
	assertValue("empty total", empty.TotalTokens, nil)
	if len(empty.Daily) != 1 || empty.Daily[0].ExecutionCount != 0 {
		t.Fatalf("empty daily = %+v", empty.Daily)
	}
	assertValue("empty daily input", empty.Daily[0].InputTokens, nil)

	distribution, err := service.ProviderUsageDistribution(ctx, analytics.RangeQuery{
		From: day, To: day, ProviderID: providers.OpenCodeGoID, ModelID: "unknown",
	})
	if err != nil || len(distribution.ByModel) != 1 {
		t.Fatalf("unknown distribution = %+v, err %v", distribution, err)
	}
	assertValue("distribution unknown input", distribution.ByModel[0].InputTokens, nil)
	combinedDistribution, err := service.ProviderUsageDistribution(ctx, analytics.RangeQuery{
		From: day, To: day, ProviderID: providers.OpenCodeGoID, ModelID: "combined",
	})
	if err != nil || len(combinedDistribution.ByProvider) != 1 || len(combinedDistribution.ByModel) != 1 {
		t.Fatalf("combined distribution = %+v, err %v", combinedDistribution, err)
	}
	assertValue("combined distribution input", combinedDistribution.ByModel[0].InputTokens, &inputSix)
	assertValue("combined distribution output", combinedDistribution.ByModel[0].OutputTokens, &outputThree)
	assertValue("combined distribution total", combinedDistribution.ByModel[0].TotalTokens, &totalFive)
	zeroDistribution, err := service.ProviderUsageDistribution(ctx, analytics.RangeQuery{
		From: day, To: day, ProviderID: providers.OpenCodeGoID, ModelID: "zero",
	})
	if err != nil || len(zeroDistribution.ByModel) != 1 {
		t.Fatalf("zero distribution = %+v, err %v", zeroDistribution, err)
	}
	assertValue("zero distribution input", zeroDistribution.ByModel[0].InputTokens, &zero)
	emptyDistribution, err := service.ProviderUsageDistribution(ctx, analytics.RangeQuery{
		From: day, To: day, ProviderID: providers.OpenCodeGoID, ModelID: "missing",
	})
	if err != nil || len(emptyDistribution.ByProvider) != 0 || len(emptyDistribution.ByModel) != 0 {
		t.Fatalf("empty provider usage distribution = %+v, err %v", emptyDistribution, err)
	}
}

func providerUsageRecord(
	executionID string,
	modelID string,
	status metering.Status,
	inputTokens *int64,
	outputTokens *int64,
	totalTokens *int64,
	day time.Time,
) metering.Record {
	return metering.Record{
		ExecutionID: executionID, ProviderID: providers.OpenCodeGoID, ModelID: modelID,
		InputTokens: inputTokens, OutputTokens: outputTokens, TotalTokens: totalTokens,
		Status: status, Source: metering.SourceOpenCodeGo, RecordedAt: day.Add(time.Hour),
	}
}
