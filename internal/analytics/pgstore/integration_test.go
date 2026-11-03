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
}
