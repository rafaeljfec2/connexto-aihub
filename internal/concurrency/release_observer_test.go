package concurrency_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"aihub/internal/concurrency"
	"aihub/internal/execution"
)

type emptyExecutionSink struct{}

func (emptyExecutionSink) Commit(int, string)      {}
func (emptyExecutionSink) WriteChunk([]byte) error { return nil }

func TestExecutionDrainCompletesAfterBoundedLeaseRelease(t *testing.T) {
	repo := &releaseAttemptsRepo{blockRelease: true, releaseStarted: make(chan struct{})}
	service := concurrency.NewService(repo, concurrency.WithOperationTimeout(20*time.Millisecond))
	openStarted := make(chan struct{})
	manager := execution.NewManager(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"provider",
		func(ctx context.Context, _ execution.Target, _ []byte, _, _ string) (*http.Response, error) {
			close(openStarted)
			<-ctx.Done()
			return nil, ctx.Err()
		},
		time.Second,
		execution.WithObserver(concurrency.NewReleaseObserver(service, nil)),
	)
	executionDone := make(chan struct{})
	go func() {
		defer close(executionDone)
		_, _ = manager.Execute(context.Background(), execution.Command{
			ExecutionID:       "exec-drain",
			Target:            execution.Target{ProviderID: "provider", ModelID: "model"},
			ProviderSessionID: "session",
			ContentType:       "application/json",
		}, emptyExecutionSink{})
	}()
	<-openStarted
	manager.BeginDrain()
	manager.CancelActive()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := time.Now()
	if err := manager.Wait(ctx); err != nil {
		t.Fatalf("Wait after bounded release = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("drain took %v, want bounded completion", elapsed)
	}
	select {
	case <-repo.releaseStarted:
	default:
		t.Fatal("lease release was not attempted")
	}
	select {
	case <-executionDone:
	case <-time.After(time.Second):
		t.Fatal("execution remained active after lease release deadline")
	}
}
