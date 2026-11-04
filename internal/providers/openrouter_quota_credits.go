package providers

import (
	"context"
	"encoding/json"
)

type openRouterCreditsEnvelope struct {
	Data openRouterCreditsData `json:"data"`
}

type openRouterCreditsData struct {
	TotalCredits *float64 `json:"total_credits"`
	TotalUsage   *float64 `json:"total_usage"`
}

// FetchAccountQuota queries account-level OpenRouter credits with a Management API Key only.
func (a *OpenRouterQuotaAdapter) FetchAccountQuota(ctx context.Context, endpoint, managementCredential string) (ProviderQuotaSnapshot, error) {
	snapshot := newOpenRouterSnapshot(ScopeAccount, "openrouter_credits_api")
	body, requestError := a.request(ctx, endpoint, "/credits", managementCredential)
	if requestError != nil {
		applyQuotaError(&snapshot, requestError)
		return snapshot, nil
	}
	var envelope openRouterCreditsEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		applyQuotaError(&snapshot, &ProviderQuotaError{Code: "json_parse_error", Message: "Formato de resposta inesperado do provedor"})
		return snapshot, nil
	}
	unit := "USD"
	appendKeyMetric(&snapshot, "total_credits", "Total account credits", NatureQuota,
		nil, envelope.Data.TotalCredits, nil, &unit, nil, nil, nil)
	appendKeyMetric(&snapshot, "total_usage", "Total account usage", NatureCost,
		envelope.Data.TotalUsage, nil, nil, &unit, nil, nil, nil)
	snapshot.Status = QuotaStatusAvailable
	return snapshot, nil
}
