package routing_test

import (
	"testing"
	"time"

	"aihub/internal/providers"
	"aihub/internal/routing"
)

func quotaFixtureOpenCode(pct float64, now time.Time) routing.AutoQuotaView {
	snap := providers.ProviderQuotaSnapshot{
		ProviderID: providers.OpenCodeGoID,
		Status:     providers.QuotaStatusAvailable,
		Source:     providers.ProviderQuotaSource{FetchedAt: now.Format(time.RFC3339)},
		Quotas:     []providers.ProviderQuotaMetric{{Metric: "go_weekly", ConsumedPercent: &pct}},
	}
	return routing.BuildAutoQuotaView([]providers.ProviderQuotaSnapshot{snap}, now, 2*time.Minute)
}

func TestSelectPoolAWRR_PrefersLowerPressure(t *testing.T) {
	now := time.Now().UTC()
	low := 10.0
	snaps := []providers.ProviderQuotaSnapshot{
		{
			ProviderID: providers.OpenCodeGoID,
			Status:     providers.QuotaStatusAvailable,
			Source:     providers.ProviderQuotaSource{FetchedAt: now.Format(time.RFC3339)},
			Quotas:     []providers.ProviderQuotaMetric{{Metric: "go_weekly", ConsumedPercent: &low}},
		},
		{
			ProviderID: providers.OpenRouterID,
			Status:     providers.QuotaStatusAvailable,
			Source:     providers.ProviderQuotaSource{FetchedAt: now.Format(time.RFC3339)},
			Quotas: []providers.ProviderQuotaMetric{
				{Metric: "limit", Limit: ptrFloat(100)},
				{Metric: "limit_remaining", Remaining: ptrFloat(20)},
			},
		},
	}
	quota := routing.BuildAutoQuotaView(snaps, now, 2*time.Minute)
	candidates := []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenRouterID, ModelID: "anthropic/claude-haiku-4.5"},
	}
	poolCounts := map[routing.AutoPoolID]int{}
	for i := 0; i < 20; i++ {
		pool, _ := routing.SelectPoolAWRR(candidates, quota)
		poolCounts[pool]++
	}
	if poolCounts[routing.PoolOpenCodeShared] <= poolCounts[routing.PoolOpenRouterKey] {
		t.Fatalf("expected more OpenCode selections, counts=%v", poolCounts)
	}
}

func TestSelectPoolAWRR_UnknownQuotaRotates(t *testing.T) {
	quota := routing.AutoQuotaView{Pools: map[routing.AutoPoolID]routing.QuotaPressureState{}}
	candidates := []routing.Target{
		{ProviderID: providers.OpenCodeGoID, ModelID: "gpt-6-luna"},
		{ProviderID: providers.OpenRouterID, ModelID: "anthropic/claude-haiku-4.5"},
	}
	seen := map[routing.AutoPoolID]bool{}
	for i := 0; i < 6; i++ {
		pool, reasons := routing.SelectPoolAWRR(candidates, quota)
		if len(reasons) == 0 || reasons[0] != "quota_unknown_all" {
			t.Fatalf("reasons = %v", reasons)
		}
		seen[pool] = true
	}
	if len(seen) < 2 {
		t.Fatalf("expected alternation across unknown pools, seen=%v", seen)
	}
}

func ptrFloat(v float64) *float64 { return &v }
