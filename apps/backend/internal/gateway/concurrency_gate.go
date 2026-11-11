package gateway

import (
	"context"

	"aihub/internal/concurrency"
)

// ConcurrencyGate performs E09.3 slot acquisition before daily resource admission.
type ConcurrencyGate interface {
	Acquire(ctx context.Context, request concurrency.AcquireRequest) (concurrency.AcquireDecision, error)
	Release(ctx context.Context, executionID string) error
	RenewLease(ctx context.Context, executionID string) error
}
