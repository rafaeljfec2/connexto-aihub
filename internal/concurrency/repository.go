package concurrency

import (
	"context"
	"time"
)

// Repository persists concurrency policies and leases.
type Repository interface {
	CreatePolicy(ctx context.Context, input NewPolicyInput) (Policy, error)
	UpdatePolicy(ctx context.Context, id int64, input UpdatePolicyInput) (Policy, error)
	ConfigurePolicy(ctx context.Context, id int64, concurrencyLimit int, enabled bool) (Policy, error)
	SetPolicyEnabled(ctx context.Context, id int64, enabled bool) (Policy, error)
	GetPolicy(ctx context.Context, id int64) (Policy, error)
	GetPolicyByScope(ctx context.Context, scope Scope) (Policy, error)
	ListPolicies(ctx context.Context) ([]Policy, error)
	ListUsage(ctx context.Context, query UsageQuery) ([]UsageSnapshot, error)

	Acquire(ctx context.Context, request AcquireRequest, now time.Time, leaseTTL time.Duration) (AcquireDecision, error)
	Release(ctx context.Context, executionID string) error
	RenewLease(ctx context.Context, executionID string, now time.Time, leaseTTL time.Duration) error
}
