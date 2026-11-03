// Package concurrency implements E09.3 simultaneous execution limits backed by PostgreSQL.
package concurrency

import (
	"time"

	"aihub/internal/resources"
)

// Scope aliases E09 resource scope keys for policy identity.
type ScopeKind = resources.ScopeKind

const (
	ScopeGlobal   = resources.ScopeGlobal
	ScopeProvider = resources.ScopeProvider
	ScopeModel    = resources.ScopeModel
)

// Scope is a policy identity key.
type Scope = resources.ResourceScope

// Policy configures a simultaneous execution ceiling.
type Policy struct {
	ID                int64
	Scope             Scope
	ConcurrencyLimit  int
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewPolicyInput creates a policy row.
type NewPolicyInput struct {
	Scope             Scope
	ConcurrencyLimit  int
	Enabled           bool
}

// UpdatePolicyInput changes limit fields.
type UpdatePolicyInput struct {
	ConcurrencyLimit *int
}

// UpsertPolicyInput upserts one scope key.
type UpsertPolicyInput struct {
	Scope             Scope
	ConcurrencyLimit  int
	Enabled           bool
}

// AcquireRequest reserves one execution slot.
type AcquireRequest struct {
	ExecutionID string
	ProviderID  string
	ModelID     string
}

// ExpiredLease is one lease row removed during acquire purge.
type ExpiredLease struct {
	ExecutionID string
	ProviderID  string
	ModelID     string
}

// AcquireDecision is the outcome of one acquire attempt.
// When no enabled policies apply, Allowed is true and no lease row is created;
// usage active_count then stays at zero for that execution.
type AcquireDecision struct {
	ExecutionID   string
	Allowed       bool
	Reason        string
	ExpiredLeases []ExpiredLease
}

// UsageSnapshot reports active leases against one policy.
type UsageSnapshot struct {
	Policy        Policy
	ActiveCount   int
	Available     int
}

// UsageQuery filters usage listing.
type UsageQuery struct {
	ProviderID string
	ModelID    string
}
