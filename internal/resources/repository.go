package resources

import (
	"context"
	"time"
)

// Repository persists policies and performs atomic admission accounting.
type Repository interface {
	CreatePolicy(ctx context.Context, input NewPolicyInput) (ResourcePolicy, error)
	UpdatePolicy(ctx context.Context, id int64, input UpdatePolicyInput) (ResourcePolicy, error)
	ConfigurePolicy(ctx context.Context, id int64, dailyLimit int, enabled bool) (ResourcePolicy, error)
	SetPolicyEnabled(ctx context.Context, id int64, enabled bool) (ResourcePolicy, error)
	GetPolicy(ctx context.Context, id int64) (ResourcePolicy, error)
	ListPolicies(ctx context.Context) ([]ResourcePolicy, error)
	GetPolicyByScope(ctx context.Context, scope ResourceScope) (ResourcePolicy, error)
	ListUsage(ctx context.Context, query UsageQuery) ([]UsageSnapshot, error)

	Admit(ctx context.Context, request AdmissionRequest, now time.Time) (AdmissionDecision, error)
}
