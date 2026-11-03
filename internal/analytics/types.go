// Package analytics implements E09.4 usage analytics over durable admission rows.
package analytics

import "time"

// RangeQuery selects admitted volume for inclusive UTC calendar days.
type RangeQuery struct {
	From       time.Time
	To         time.Time
	ProviderID string
	ModelID    string
}

// DailyAdmissionCount is admitted volume for one UTC day.
type DailyAdmissionCount struct {
	PeriodStart   time.Time
	AdmittedCount int
}

// ProviderDailyCount attributes admissions to one provider on one UTC day.
type ProviderDailyCount struct {
	PeriodStart   time.Time
	ProviderID    string
	AdmittedCount int
}

// ModelDailyCount attributes admissions to one provider/model on one UTC day.
type ModelDailyCount struct {
	PeriodStart   time.Time
	ProviderID    string
	ModelID       string
	AdmittedCount int
}

// AdmissionsReport is the global admitted volume for a range.
type AdmissionsReport struct {
	From          time.Time
	To            time.Time
	TotalAdmitted int
	Daily         []DailyAdmissionCount
}

// DistributionReport breaks down admissions by provider and model per UTC day.
type DistributionReport struct {
	From       time.Time
	To         time.Time
	ByProvider []ProviderDailyCount
	ByModel    []ModelDailyCount
}

// ProviderUsageDailyTokens aggregates token sums for one UTC day.
type ProviderUsageDailyTokens struct {
	PeriodStart    time.Time
	InputTokens    int64
	OutputTokens   int64
	TotalTokens    int64
	ExecutionCount int
}

// ProviderUsageSummary aggregates persisted provider usage for a UTC range.
type ProviderUsageSummary struct {
	From                  time.Time
	To                    time.Time
	InputTokens           int64
	OutputTokens          int64
	TotalTokens           int64
	ReportedExecutions    int
	UnavailableExecutions int
	PartialExecutions     int
	InvalidExecutions     int
	Daily                 []ProviderUsageDailyTokens
}

// ProviderUsageProviderDaily attributes token sums to one provider per UTC day.
type ProviderUsageProviderDaily struct {
	PeriodStart  time.Time
	ProviderID   string
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	Executions   int
}

// ProviderUsageModelDaily attributes token sums to one provider/model per UTC day.
type ProviderUsageModelDaily struct {
	PeriodStart  time.Time
	ProviderID   string
	ModelID      string
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
	Executions   int
}

// ProviderUsageDistributionReport breaks down provider usage by provider and model.
type ProviderUsageDistributionReport struct {
	From       time.Time
	To         time.Time
	ByProvider []ProviderUsageProviderDaily
	ByModel    []ProviderUsageModelDaily
}
