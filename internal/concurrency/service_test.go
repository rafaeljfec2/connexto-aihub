package concurrency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"aihub/internal/concurrency"
)

type releaseAttemptsRepo struct {
	attempts       int
	failures       int
	blockRelease   bool
	blockRenew     bool
	releaseStarted chan struct{}
	renewStarted   chan struct{}
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
func (r *releaseAttemptsRepo) Release(ctx context.Context, _ string) error {
	r.attempts++
	if r.blockRelease {
		if r.releaseStarted != nil {
			close(r.releaseStarted)
		}
		<-ctx.Done()
		return ctx.Err()
	}
	if r.attempts <= r.failures {
		return concurrency.ErrUnavailable
	}
	return nil
}
func (r *releaseAttemptsRepo) RenewLease(ctx context.Context, _ string, _ time.Time, _ time.Duration) error {
	if r.blockRenew {
		if r.renewStarted != nil {
			close(r.renewStarted)
		}
		<-ctx.Done()
		return ctx.Err()
	}
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

func TestReleaseStopsAtOperationTimeout(t *testing.T) {
	repo := &releaseAttemptsRepo{blockRelease: true, releaseStarted: make(chan struct{})}
	service := concurrency.NewService(repo, concurrency.WithOperationTimeout(20*time.Millisecond))
	started := time.Now()
	err := service.Release(context.Background(), "exec-timeout")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Release = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Release took %v, want bounded completion", elapsed)
	}
	select {
	case <-repo.releaseStarted:
	default:
		t.Fatal("repository release was not called")
	}
}

func TestReleasePropagatesCallerCancellation(t *testing.T) {
	repo := &releaseAttemptsRepo{blockRelease: true, releaseStarted: make(chan struct{})}
	service := concurrency.NewService(repo)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- service.Release(ctx, "exec-canceled") }()
	<-repo.releaseStarted
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Release = %v, want canceled", err)
	}
}

func TestRenewLeaseStopsAtOperationTimeout(t *testing.T) {
	repo := &releaseAttemptsRepo{blockRenew: true, renewStarted: make(chan struct{})}
	service := concurrency.NewService(repo, concurrency.WithOperationTimeout(20*time.Millisecond))
	err := service.RenewLease(context.Background(), "exec-renew-timeout")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RenewLease = %v, want deadline exceeded", err)
	}
	select {
	case <-repo.renewStarted:
	default:
		t.Fatal("repository renewal was not called")
	}
}
