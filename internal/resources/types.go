// Package resources implements E09 resource admission: durable daily request
// ceilings checked after routing and before execution starts.
package resources

import "time"

// ScopeKind identifies which dimension a policy applies to.
type ScopeKind string

const (
	ScopeGlobal   ScopeKind = "global"
	ScopeProvider ScopeKind = "provider"
	ScopeModel    ScopeKind = "model"
)

// ResourceScope is the identity of one policy row.
type ResourceScope struct {
	Kind       ScopeKind
	ProviderID string
	ModelID    string
}

// ResourcePolicy is a configured daily admission ceiling.
type ResourcePolicy struct {
	ID         int64
	Scope      ResourceScope
	DailyLimit int
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewPolicyInput creates a policy row.
type NewPolicyInput struct {
	Scope      ResourceScope
	DailyLimit int
	Enabled    bool
}

// UpdatePolicyInput changes a policy. Nil fields are unchanged.
type UpdatePolicyInput struct {
	DailyLimit *int
}

// UpsertPolicyInput creates or replaces configuration for one scope key.
type UpsertPolicyInput struct {
	Scope      ResourceScope
	DailyLimit int
	Enabled    bool
}

// UsageQuery selects usage rows for one UTC period.
type UsageQuery struct {
	PeriodStart time.Time
	ProviderID  string
	ModelID     string
}

// UsageSnapshot is admitted volume for one policy period.
type UsageSnapshot struct {
	Policy        ResourcePolicy
	PeriodStart   time.Time
	ResetAt       time.Time
	AdmittedCount int
}

// AdmissionRequest reserves one Hub admission unit for a routed target.
type AdmissionRequest struct {
	// ID is the stable internal admission identifier (ADR-023). When empty,
	// Service generates one before the first persistence attempt.
	ID         string
	ProviderID string
	ModelID    string
}

// AdmissionDecision is the outcome of one admission attempt.
type AdmissionDecision struct {
	AdmissionID  string
	Allowed      bool
	Reason       string
	PeriodStart  time.Time
	ResetAt      time.Time
	GlobalCount  int
	ProviderCount int
	ModelCount   int
}

// UsagePeriodBounds returns the UTC calendar day containing instant.
func UsagePeriodBounds(instant time.Time) (start time.Time, resetAt time.Time) {
	utc := instant.UTC()
	start = time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	resetAt = start.Add(24 * time.Hour)
	return start, resetAt
}
