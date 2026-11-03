package concurrency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"aihub/internal/concurrency"
)

type releaseAttemptsRepo struct {
	attempts int
	failures int
}

func (r *releaseAttemptsRepo) CreatePolicy(context.Context, concurrency.NewPolicyInput) (concurrency.Policy, error) {
	return concurrency.Policy{}, concurrency.ErrNotFound
}
func (r *releaseAttemptsRepo) UpdatePolicy(context.Context, int64, concurrency.UpdatePolicyInput) (concurrency.Policy, error) {
	return concurrency.Policy{}, concurrency.ErrNotFound
}
func (r *releaseAttemptsRepo) ConfigurePolicy(context.Context, int64, int, bool) (concurrency.Policy, error) {
	return concurrency.Policy{}, concurrency.ErrNotFound
}
func (r *releaseAttemptsRepo) SetPolicyEnabled(context.Context, int64, bool) (concurrency.Policy, error) {
	return concurrency.Policy{}, concurrency.ErrNotFound
}
func (r *releaseAttemptsRepo) GetPolicy(context.Context, int64) (concurrency.Policy, error) {
	return concurrency.Policy{}, concurrency.ErrNotFound
}
func (r *releaseAttemptsRepo) GetPolicyByScope(context.Context, concurrency.Scope) (concurrency.Policy, error) {
	return concurrency.Policy{}, concurrency.ErrNotFound
}
func (r *releaseAttemptsRepo) ListPolicies(context.Context) ([]concurrency.Policy, error) {
	return nil, nil
}
func (r *releaseAttemptsRepo) ListUsage(context.Context, concurrency.UsageQuery) ([]concurrency.UsageSnapshot, error) {
	return nil, nil
}
func (r *releaseAttemptsRepo) Acquire(context.Context, concurrency.AcquireRequest, time.Time, time.Duration) (concurrency.AcquireDecision, error) {
	return concurrency.AcquireDecision{}, nil
}
func (r *releaseAttemptsRepo) Release(context.Context, string) error {
	r.attempts++
	if r.attempts <= r.failures {
		return concurrency.ErrUnavailable
	}
	return nil
}
func (r *releaseAttemptsRepo) RenewLease(context.Context, string, time.Time, time.Duration) error {
	return nil
}

func TestReleaseRetriesTransientFailures(t *testing.T) {
	repo := &releaseAttemptsRepo{failures: 2}
	service := concurrency.NewService(repo)
	if err := service.Release(context.Background(), "exec-1"); err != nil {
		t.Fatalf("Release = %v, want success after retries", err)
	}
	if repo.attempts != 3 {
		t.Fatalf("attempts = %d, want 3", repo.attempts)
	}
}

func TestReleaseReturnsLastErrorAfterRetries(t *testing.T) {
	repo := &releaseAttemptsRepo{failures: 10}
	err := concurrency.NewService(repo).Release(context.Background(), "exec-1")
	if !errors.Is(err, concurrency.ErrUnavailable) {
		t.Fatalf("Release = %v, want ErrUnavailable", err)
	}
	if repo.attempts != 5 {
		t.Fatalf("attempts = %d, want 5", repo.attempts)
	}
}
