package routing_test

import (
	"testing"
	"time"

	"aihub/internal/providers"
	"aihub/internal/routing"
)

func TestBuildAutoQuotaView_OpenCodePressure(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	rolling := 40.0
	weekly := 55.0
	monthly := 20.0
	snap := providers.ProviderQuotaSnapshot{
		ProviderID: providers.OpenCodeGoID,
		Status:     providers.QuotaStatusAvailable,
		Source:     providers.ProviderQuotaSource{FetchedAt: now.Format(time.RFC3339)},
		Quotas: []providers.ProviderQuotaMetric{
			{Metric: "go_rolling", ConsumedPercent: &rolling},
			{Metric: "go_weekly", ConsumedPercent: &weekly},
			{Metric: "go_monthly", ConsumedPercent: &monthly},
		},
	}
	view := routing.BuildAutoQuotaView([]providers.ProviderQuotaSnapshot{snap}, now, 2*time.Minute)
	st := view.Pools[routing.PoolOpenCodeShared]
	if st.Pressure == nil || *st.Pressure != 0.55 {
		t.Fatalf("pressure = %v, want 0.55", st.Pressure)
	}
}

func TestBuildAutoQuotaView_OpenRouterKey(t *testing.T) {
	now := time.Now().UTC()
	limit := 100.0
	rem := 25.0
	snap := providers.ProviderQuotaSnapshot{
		ProviderID: providers.OpenRouterID,
		Status:     providers.QuotaStatusAvailable,
		Source:     providers.ProviderQuotaSource{FetchedAt: now.Format(time.RFC3339)},
		Quotas: []providers.ProviderQuotaMetric{
			{Metric: "limit", Limit: &limit},
			{Metric: "limit_remaining", Remaining: &rem},
		},
	}
	view := routing.BuildAutoQuotaView([]providers.ProviderQuotaSnapshot{snap}, now, 2*time.Minute)
	st := view.Pools[routing.PoolOpenRouterKey]
	if st.Pressure == nil || *st.Pressure != 0.75 {
		t.Fatalf("pressure = %v, want 0.75", st.Pressure)
	}
}

func TestPoolIDForTarget_OpenCodeShared(t *testing.T) {
	id := routing.PoolIDForTarget(routing.Target{ProviderID: providers.OpenCodeGoID, ModelID: "glm-5.3-flash"})
	if id != routing.PoolOpenCodeShared {
		t.Fatalf("pool = %q", id)
	}
}
