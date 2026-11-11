package gateway

import (
	"net/http"
)

// ProviderUsage handles GET /api/resource-analytics/provider-usage.
func (a *AnalyticsAdmin) ProviderUsage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, err := analyticsRangeFromRequest(r, "provider_id", "model_id")
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		report, err := a.analytics.ProviderUsage(r.Context(), query)
		if err != nil {
			a.writeError(w, err)
			return
		}
		daily := make([]map[string]any, 0, len(report.Daily))
		for _, row := range report.Daily {
			daily = append(daily, map[string]any{
				"period_start":    row.PeriodStart.UTC(),
				"input_tokens":    row.InputTokens,
				"output_tokens":   row.OutputTokens,
				"total_tokens":    row.TotalTokens,
				"execution_count": row.ExecutionCount,
			})
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{
			"from":                   report.From.UTC(),
			"to":                     report.To.UTC(),
			"input_tokens":           report.InputTokens,
			"output_tokens":          report.OutputTokens,
			"total_tokens":           report.TotalTokens,
			"reported_executions":    report.ReportedExecutions,
			"unavailable_executions": report.UnavailableExecutions,
			"partial_executions":     report.PartialExecutions,
			"invalid_executions":     report.InvalidExecutions,
			"daily":                  daily,
		})
	})
}

// ProviderUsageDistribution handles GET /api/resource-analytics/provider-usage/distribution.
func (a *AnalyticsAdmin) ProviderUsageDistribution() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, err := analyticsRangeFromRequest(r, "provider_id", "model_id")
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		report, err := a.analytics.ProviderUsageDistribution(r.Context(), query)
		if err != nil {
			a.writeError(w, err)
			return
		}
		byProvider := make([]map[string]any, 0, len(report.ByProvider))
		for _, row := range report.ByProvider {
			byProvider = append(byProvider, map[string]any{
				"period_start":  row.PeriodStart.UTC(),
				"provider_id":   row.ProviderID,
				"input_tokens":  row.InputTokens,
				"output_tokens": row.OutputTokens,
				"total_tokens":  row.TotalTokens,
				"executions":    row.Executions,
			})
		}
		byModel := make([]map[string]any, 0, len(report.ByModel))
		for _, row := range report.ByModel {
			byModel = append(byModel, map[string]any{
				"period_start":  row.PeriodStart.UTC(),
				"provider_id":   row.ProviderID,
				"model_id":      row.ModelID,
				"input_tokens":  row.InputTokens,
				"output_tokens": row.OutputTokens,
				"total_tokens":  row.TotalTokens,
				"executions":    row.Executions,
			})
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{
			"from":        report.From.UTC(),
			"to":          report.To.UTC(),
			"by_provider": byProvider,
			"by_model":    byModel,
		})
	})
}

// Concurrency handles GET /api/resource-analytics/concurrency.
