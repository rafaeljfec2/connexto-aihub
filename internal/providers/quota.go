package providers

import (
	"context"
)

// ProviderQuotaStatus describes the operational outcome of a quota query.
type ProviderQuotaStatus string

const (
	QuotaStatusAvailable              ProviderQuotaStatus = "available"
	QuotaStatusUnsupported            ProviderQuotaStatus = "unsupported"
	QuotaStatusUnconfiguredCredential ProviderQuotaStatus = "unconfigured_credential"
	QuotaStatusUnauthorized           ProviderQuotaStatus = "unauthorized"
	QuotaStatusRateLimited            ProviderQuotaStatus = "rate_limited"
	QuotaStatusTimeout                ProviderQuotaStatus = "timeout"
	QuotaStatusError                  ProviderQuotaStatus = "error"
)

// DataScope defines the organizational breadth of the reported quota data.
type DataScope string

const (
	ScopeAPIKey       DataScope = "api_key"
	ScopeAccount      DataScope = "account"
	ScopeWorkspace    DataScope = "workspace"
	ScopeOrganization DataScope = "organization"
	ScopeSubscription DataScope = "subscription"
	ScopeUnknown      DataScope = "unknown"
)

// AccountQuotaAdapter reads provider-level balances requiring administrative credentials.
type AccountQuotaAdapter interface {
	FetchAccountQuota(ctx context.Context, endpoint, managementCredential string) (ProviderQuotaSnapshot, error)
}

// MetricNature specifies the operational classification of each metric.
type MetricNature string

const (
	NatureQuota     MetricNature = "quota"
	NatureUsage     MetricNature = "usage"
	NatureCost      MetricNature = "cost"
	NatureRateLimit MetricNature = "rate_limit"
)

// ProviderQuotaMetric represents an individual metric returned by the provider.
// No arithmetic calculations or percentage derivations are made on missing fields.
type ProviderQuotaMetric struct {
	Metric           string       `json:"metric"`
	Label            string       `json:"label"`
	Nature           MetricNature `json:"nature"`
	Used             *float64     `json:"used,omitempty"`
	Limit            *float64     `json:"limit,omitempty"`
	Remaining        *float64     `json:"remaining,omitempty"`
	ConsumedPercent  *float64     `json:"consumed_percent,omitempty"`
	RemainingPercent *float64     `json:"remaining_percent,omitempty"`
	Unit             *string      `json:"unit,omitempty"`
	Period           *string      `json:"period,omitempty"`
	StartsAt         *string      `json:"starts_at,omitempty"`
	EndsAt           *string      `json:"ends_at,omitempty"`
	ResetsAt         *string      `json:"resets_at,omitempty"`
	Scope            string       `json:"scope,omitempty"`
	IncludeBYOK      *bool        `json:"include_byok,omitempty"`
}

// ProviderQuotaSource documents provenance, scope, and timestamps.
type ProviderQuotaSource struct {
	Type               string    `json:"type"`
	Name               string    `json:"name"`
	Scope              DataScope `json:"scope"`
	FetchedAt          string    `json:"fetched_at"`
	ProviderReportedAt *string   `json:"provider_reported_at,omitempty"`
}

// ProviderQuotaError contains classified failure details.
type ProviderQuotaError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ProviderUsageChartPoint is one provider-reported daily cost bar.
type ProviderUsageChartPoint struct {
	Date           string  `json:"date"`
	CostMicroCents int64   `json:"cost_micro_cents"`
	CostUSD        float64 `json:"cost_usd"`
	Tokens         *int64  `json:"tokens,omitempty"`
	Requests       *int64  `json:"requests,omitempty"`
}

// ProviderUsageChart holds the official workspace usage series for the drawer chart.
type ProviderUsageChart struct {
	Title           string                    `json:"title"`
	Subtitle        string                    `json:"subtitle"`
	TotalMicroCents int64                     `json:"total_micro_cents"`
	TotalUSD        float64                   `json:"total_usd"`
	Unit            string                    `json:"unit"`
	PeriodStart     string                    `json:"period_start"`
	PeriodEnd       string                    `json:"period_end"`
	Points          []ProviderUsageChartPoint `json:"points"`
}

// ProviderQuotaSnapshot holds the normalized official reading for an account.
type ProviderQuotaSnapshot struct {
	AccountID         string                `json:"account_id"`
	AccountLabel      string                `json:"account_label"`
	ProviderID        string                `json:"provider_id"`
	ProviderName      string                `json:"provider_name"`
	Status            ProviderQuotaStatus   `json:"status"`
	Scope             DataScope             `json:"scope"`
	Quotas            []ProviderQuotaMetric `json:"quotas"`
	Source            ProviderQuotaSource   `json:"source"`
	Error             *ProviderQuotaError   `json:"error,omitempty"`
	ProviderExpiresAt *string               `json:"provider_expires_at,omitempty"`
	WorkspaceID       *string               `json:"workspace_id,omitempty"`
	WorkspaceName     *string               `json:"workspace_name,omitempty"`
	UsageURL          *string               `json:"usage_url,omitempty"`
	UsageChart        *ProviderUsageChart   `json:"usage_chart,omitempty"`
}

// ProviderQuotasResponse wraps all account snapshots returned by the query.
type ProviderQuotasResponse struct {
	Snapshots []ProviderQuotaSnapshot `json:"snapshots"`
	FetchedAt string                  `json:"fetched_at"`
}

// QuotaAdapter queries the provider's official API to retrieve quota details.
type QuotaAdapter interface {
	FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error)
}
