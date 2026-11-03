package gateway

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"aihub/internal/analytics"
	"aihub/internal/concurrency"
	"aihub/internal/resources"
)

const (
	analyticsAdminInternalMsg = "the resource analytics request failed"
)

// AnalyticsAdmin serves E09.4 analytics endpoints (PostgreSQL mode only).
type AnalyticsAdmin struct {
	analytics   *analytics.Service
	resources   *resources.Service
	concurrency *concurrency.Service
	logger      *slog.Logger
}

// NewAnalyticsAdmin builds analytics handlers.
func NewAnalyticsAdmin(analyticsService *analytics.Service, resourcesService *resources.Service,
	concurrencyService *concurrency.Service, logger *slog.Logger) *AnalyticsAdmin {
	return &AnalyticsAdmin{
		analytics: analyticsService, resources: resourcesService,
		concurrency: concurrencyService, logger: logger,
	}
}

type analyticsDailyResponse struct {
	PeriodStart   time.Time `json:"period_start"`
	AdmittedCount int       `json:"admitted_count"`
}

type analyticsAdmissionsResponse struct {
	From          time.Time                `json:"from"`
	To            time.Time                `json:"to"`
	TotalAdmitted int                      `json:"total_admitted"`
	Daily         []analyticsDailyResponse `json:"daily"`
}

type analyticsProviderDailyResponse struct {
	PeriodStart   time.Time `json:"period_start"`
	ProviderID    string    `json:"provider_id"`
	AdmittedCount int       `json:"admitted_count"`
}

type analyticsModelDailyResponse struct {
	PeriodStart   time.Time `json:"period_start"`
	ProviderID    string    `json:"provider_id"`
	ModelID       string    `json:"model_id"`
	AdmittedCount int       `json:"admitted_count"`
}

type analyticsDistributionResponse struct {
	From       time.Time                        `json:"from"`
	To         time.Time                        `json:"to"`
	ByProvider []analyticsProviderDailyResponse `json:"by_provider"`
	ByModel    []analyticsModelDailyResponse    `json:"by_model"`
}

type analyticsQuotaUtilizationResponse struct {
	PolicyID           int64     `json:"policy_id"`
	Scope              string    `json:"scope"`
	ProviderID         string    `json:"provider_id,omitempty"`
	ModelID            string    `json:"model_id,omitempty"`
	DailyLimit         int       `json:"daily_limit"`
	Enabled            bool      `json:"enabled"`
	PeriodStart        time.Time `json:"period_start"`
	ResetAt            time.Time `json:"reset_at"`
	AdmittedCount      int       `json:"admitted_count"`
	RemainingRequests  int       `json:"remaining_requests"`
	UtilizationPercent float64   `json:"utilization_percent"`
}

type analyticsConcurrencyResponse struct {
	PolicyID         int64  `json:"policy_id"`
	Scope            string `json:"scope"`
	ProviderID       string `json:"provider_id,omitempty"`
	ModelID          string `json:"model_id,omitempty"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
	Enabled          bool   `json:"enabled"`
	ActiveCount      int    `json:"active_count"`
	Available        int    `json:"available"`
}

// Admissions handles GET /api/resource-analytics/admissions.
func (a *AnalyticsAdmin) Admissions() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, err := analyticsRangeFromRequest(r, "provider_id", "model_id")
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		report, err := a.analytics.Admissions(r.Context(), query)
		if err != nil {
			a.writeError(w, err)
			return
		}
		daily := make([]analyticsDailyResponse, 0, len(report.Daily))
		for _, row := range report.Daily {
			daily = append(daily, analyticsDailyResponse{
				PeriodStart: row.PeriodStart.UTC(), AdmittedCount: row.AdmittedCount,
			})
		}
		writeAdminJSON(w, http.StatusOK, analyticsAdmissionsResponse{
			From: report.From.UTC(), To: report.To.UTC(),
			TotalAdmitted: report.TotalAdmitted, Daily: daily,
		})
	})
}

// Distribution handles GET /api/resource-analytics/admissions/distribution.
func (a *AnalyticsAdmin) Distribution() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, err := analyticsRangeFromRequest(r, "provider_id", "model_id")
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		report, err := a.analytics.Distribution(r.Context(), query)
		if err != nil {
			a.writeError(w, err)
			return
		}
		byProvider := make([]analyticsProviderDailyResponse, 0, len(report.ByProvider))
		for _, row := range report.ByProvider {
			byProvider = append(byProvider, analyticsProviderDailyResponse{
				PeriodStart: row.PeriodStart.UTC(), ProviderID: row.ProviderID, AdmittedCount: row.AdmittedCount,
			})
		}
		byModel := make([]analyticsModelDailyResponse, 0, len(report.ByModel))
		for _, row := range report.ByModel {
			byModel = append(byModel, analyticsModelDailyResponse{
				PeriodStart: row.PeriodStart.UTC(), ProviderID: row.ProviderID,
				ModelID: row.ModelID, AdmittedCount: row.AdmittedCount,
			})
		}
		writeAdminJSON(w, http.StatusOK, analyticsDistributionResponse{
			From: report.From.UTC(), To: report.To.UTC(),
			ByProvider: byProvider, ByModel: byModel,
		})
	})
}

// QuotaUtilization handles GET /api/resource-analytics/quota-utilization.
func (a *AnalyticsAdmin) QuotaUtilization() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := rejectUnknownQueryParams(r, "provider_id", "model_id", "period_start"); err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		query := resources.UsageQuery{
			ProviderID: strings.TrimSpace(r.URL.Query().Get("provider_id")),
			ModelID:    strings.TrimSpace(r.URL.Query().Get("model_id")),
		}
		if raw := strings.TrimSpace(r.URL.Query().Get("period_start")); raw != "" {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				writeAdminError(w, http.StatusBadRequest, invalidRequestType, "period_start must be an RFC3339 timestamp")
				return
			}
			query.PeriodStart = parsed.UTC()
		}
		snapshots, err := a.resources.ListUsage(r.Context(), query)
		if err != nil {
			a.writeResourceError(w, err)
			return
		}
		out := make([]analyticsQuotaUtilizationResponse, 0, len(snapshots))
		for _, snapshot := range snapshots {
			out = append(out, analyticsQuotaUtilizationResponse{
				PolicyID: snapshot.Policy.ID, Scope: string(snapshot.Policy.Scope.Kind),
				ProviderID: snapshot.Policy.Scope.ProviderID, ModelID: snapshot.Policy.Scope.ModelID,
				DailyLimit: snapshot.Policy.DailyLimit, Enabled: snapshot.Policy.Enabled,
				PeriodStart: snapshot.PeriodStart.UTC(), ResetAt: snapshot.ResetAt.UTC(),
				AdmittedCount: snapshot.AdmittedCount,
				RemainingRequests: remainingRequests(snapshot.Policy, snapshot.AdmittedCount),
				UtilizationPercent: quotaUtilizationPercent(snapshot.Policy, snapshot.AdmittedCount),
			})
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"quota_utilization": out})
	})
}

// Concurrency handles GET /api/resource-analytics/concurrency.
func (a *AnalyticsAdmin) Concurrency() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := rejectUnknownQueryParams(r, "provider_id", "model_id"); err != nil {
			writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
			return
		}
		query := concurrency.UsageQuery{
			ProviderID: strings.TrimSpace(r.URL.Query().Get("provider_id")),
			ModelID:    strings.TrimSpace(r.URL.Query().Get("model_id")),
		}
		snapshots, err := a.concurrency.ListUsage(r.Context(), query)
		if err != nil {
			a.writeConcurrencyError(w, err)
			return
		}
		out := make([]analyticsConcurrencyResponse, 0, len(snapshots))
		for _, snapshot := range snapshots {
			out = append(out, analyticsConcurrencyResponse{
				PolicyID: snapshot.Policy.ID, Scope: string(snapshot.Policy.Scope.Kind),
				ProviderID: snapshot.Policy.Scope.ProviderID, ModelID: snapshot.Policy.Scope.ModelID,
				ConcurrencyLimit: snapshot.Policy.ConcurrencyLimit, Enabled: snapshot.Policy.Enabled,
				ActiveCount: snapshot.ActiveCount, Available: snapshot.Available,
			})
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"concurrency": out})
	})
}

func analyticsRangeFromRequest(r *http.Request, extraAllowed ...string) (analytics.RangeQuery, error) {
	allowed := append([]string{"from", "to"}, extraAllowed...)
	if err := rejectUnknownQueryParams(r, allowed...); err != nil {
		return analytics.RangeQuery{}, err
	}
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromRaw == "" || toRaw == "" {
		return analytics.RangeQuery{}, errors.New("from and to are required")
	}
	from, err := parseAnalyticsInstant(fromRaw)
	if err != nil {
		return analytics.RangeQuery{}, errors.New("from must be RFC3339 or YYYY-MM-DD")
	}
	to, err := parseAnalyticsInstant(toRaw)
	if err != nil {
		return analytics.RangeQuery{}, errors.New("to must be RFC3339 or YYYY-MM-DD")
	}
	query := analytics.RangeQuery{From: from, To: to}
	if len(extraAllowed) > 0 {
		query.ProviderID = strings.TrimSpace(r.URL.Query().Get("provider_id"))
		query.ModelID = strings.TrimSpace(r.URL.Query().Get("model_id"))
	}
	return query, nil
}

func parseAnalyticsInstant(raw string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func quotaUtilizationPercent(policy resources.ResourcePolicy, admitted int) float64 {
	if !policy.Enabled || policy.DailyLimit <= 0 {
		return 0
	}
	percent := float64(admitted) * 100 / float64(policy.DailyLimit)
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	return math.Round(percent*100) / 100
}

func (a *AnalyticsAdmin) writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, analytics.ErrInvalidInput) {
		writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
		return
	}
	a.logger.Error("resource analytics failed", "error", err.Error())
	writeAdminError(w, http.StatusInternalServerError, internalErrorType, analyticsAdminInternalMsg)
}

func (a *AnalyticsAdmin) writeResourceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, resources.ErrInvalidInput):
		writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
	default:
		a.logger.Error("resource analytics quota query failed", "error", err.Error())
		writeAdminError(w, http.StatusInternalServerError, internalErrorType, analyticsAdminInternalMsg)
	}
}

func (a *AnalyticsAdmin) writeConcurrencyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, concurrency.ErrInvalidInput):
		writeAdminError(w, http.StatusBadRequest, invalidRequestType, err.Error())
	default:
		a.logger.Error("resource analytics concurrency query failed", "error", err.Error())
		writeAdminError(w, http.StatusInternalServerError, internalErrorType, analyticsAdminInternalMsg)
	}
}
