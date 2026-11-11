package providers

import (
	"context"
	"encoding/json"
	"fmt"
)

type openRouterKeyEnvelope struct {
	Data openRouterKeyData `json:"data"`
}

type openRouterKeyData struct {
	Usage                  *float64                 `json:"usage"`
	Limit                  *float64                 `json:"limit"`
	LimitRemaining         *float64                 `json:"limit_remaining"`
	LimitReset             *string                  `json:"limit_reset"`
	UsageDaily             *float64                 `json:"usage_daily"`
	UsageWeekly            *float64                 `json:"usage_weekly"`
	UsageMonthly           *float64                 `json:"usage_monthly"`
	ByokUsage              *float64                 `json:"byok_usage"`
	ByokUsageDaily         *float64                 `json:"byok_usage_daily"`
	ByokUsageWeekly        *float64                 `json:"byok_usage_weekly"`
	ByokUsageMonthly       *float64                 `json:"byok_usage_monthly"`
	Budgets                []openRouterBudget       `json:"budgets"`
	FreeModelDailyRequests *openRouterRequestBudget `json:"free_model_daily_requests"`
	ExpiresAt              *string                  `json:"expires_at"`
}

type openRouterBudget struct {
	LimitUSD      *float64 `json:"limit_usd"`
	SpendUSD      *float64 `json:"spend_usd"`
	RemainingUSD  *float64 `json:"remaining_usd"`
	ResetInterval *string  `json:"reset_interval"`
	ResetsAt      *string  `json:"resets_at"`
	Scope         *string  `json:"scope"`
	IncludeBYOK   *bool    `json:"include_byok"`
}

type openRouterRequestBudget struct {
	Limit     *float64 `json:"limit"`
	Used      *float64 `json:"used"`
	Remaining *float64 `json:"remaining"`
}

// FetchQuota queries key-level OpenRouter usage with the inference API key only.
func (a *OpenRouterQuotaAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	snapshot := newOpenRouterSnapshot(ScopeAPIKey, "openrouter_key_api")
	body, requestError := a.request(ctx, endpoint, "/key", credential)
	if requestError != nil {
		applyQuotaError(&snapshot, requestError)
		return snapshot, nil
	}
	var envelope openRouterKeyEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		applyQuotaError(&snapshot, &ProviderQuotaError{Code: "json_parse_error", Message: "Formato de resposta inesperado do provedor"})
		return snapshot, nil
	}
	mapOpenRouterKeyData(&snapshot, envelope.Data)
	snapshot.Status = QuotaStatusAvailable
	return snapshot, nil
}

func mapOpenRouterKeyData(snapshot *ProviderQuotaSnapshot, data openRouterKeyData) {
	usd := "USD"
	appendKeyMetric(snapshot, "limit", "Key limit", NatureQuota, nil, data.Limit, nil, &usd, data.LimitReset, nil, nil)
	appendKeyMetric(snapshot, "limit_remaining", "Key remaining", NatureQuota, nil, nil, data.LimitRemaining, &usd, data.LimitReset, nil, nil)
	appendKeyMetric(snapshot, "usage", "Total usage", NatureCost, data.Usage, nil, nil, &usd, nil, nil, nil)
	appendKeyMetric(snapshot, "usage_daily", "Daily usage", NatureCost, data.UsageDaily, nil, nil, &usd, stringPointer("daily"), nil, nil)
	appendKeyMetric(snapshot, "usage_weekly", "Weekly usage", NatureCost, data.UsageWeekly, nil, nil, &usd, stringPointer("weekly"), nil, nil)
	appendKeyMetric(snapshot, "usage_monthly", "Monthly usage", NatureCost, data.UsageMonthly, nil, nil, &usd, stringPointer("monthly"), nil, nil)
	appendKeyMetric(snapshot, "byok_usage", "Total BYOK usage", NatureCost, data.ByokUsage, nil, nil, &usd, nil, nil, nil)
	appendKeyMetric(snapshot, "byok_usage_daily", "Daily BYOK usage", NatureCost, data.ByokUsageDaily, nil, nil, &usd, stringPointer("daily"), nil, nil)
	appendKeyMetric(snapshot, "byok_usage_weekly", "Weekly BYOK usage", NatureCost, data.ByokUsageWeekly, nil, nil, &usd, stringPointer("weekly"), nil, nil)
	appendKeyMetric(snapshot, "byok_usage_monthly", "Monthly BYOK usage", NatureCost, data.ByokUsageMonthly, nil, nil, &usd, stringPointer("monthly"), nil, nil)
	for index, budget := range data.Budgets {
		snapshot.Quotas = append(snapshot.Quotas, budgetMetric(fmt.Sprintf("budget_%d", index+1), "Budget", budget))
	}
	if data.FreeModelDailyRequests != nil {
		requests := "requests"
		appendKeyMetric(snapshot, "free_model_daily_requests", "Free model daily requests", NatureQuota,
			data.FreeModelDailyRequests.Used, data.FreeModelDailyRequests.Limit, data.FreeModelDailyRequests.Remaining,
			&requests, stringPointer("daily"), nil, nil)
	}
	snapshot.ProviderExpiresAt = data.ExpiresAt
}

func budgetMetric(metric, label string, budget openRouterBudget) ProviderQuotaMetric {
	unit := "USD"
	result := ProviderQuotaMetric{
		Metric: metric, Label: label, Nature: NatureQuota,
		Used: budget.SpendUSD, Limit: budget.LimitUSD, Remaining: budget.RemainingUSD,
		Unit: &unit, Period: budget.ResetInterval, ResetsAt: budget.ResetsAt,
		IncludeBYOK: budget.IncludeBYOK,
	}
	if budget.Scope != nil {
		result.Scope = *budget.Scope
	}
	return result
}

func appendKeyMetric(snapshot *ProviderQuotaSnapshot, metric, label string, nature MetricNature,
	used, limit, remaining *float64, unit *string, period, resetsAt, endsAt *string) {
	if used == nil && limit == nil && remaining == nil {
		return
	}
	snapshot.Quotas = append(snapshot.Quotas, ProviderQuotaMetric{
		Metric: metric, Label: label, Nature: nature, Used: used, Limit: limit,
		Remaining: remaining, Unit: unit, Period: period, ResetsAt: resetsAt, EndsAt: endsAt,
	})
}

func stringPointer(value string) *string { return &value }
