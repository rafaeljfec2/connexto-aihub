package metering

import (
	"context"
	"errors"
	"testing"
	"time"

	"aihub/internal/metering/capture"
	"aihub/internal/providers"
)

type repositoryFunc func(context.Context, Record) (bool, error)

func (f repositoryFunc) Insert(ctx context.Context, record Record) (bool, error) {
	return f(ctx, record)
}

func TestRecordFromCaptureDetachesCanceledContextAndAddsDeadline(t *testing.T) {
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	service := NewService(repositoryFunc(func(ctx context.Context, _ Record) (bool, error) {
		called = true
		if ctx.Err() != nil {
			t.Fatalf("persistence context inherited cancellation: %v", ctx.Err())
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("persistence context has no deadline")
		}
		return true, nil
	}), WithPersistenceTimeout(100*time.Millisecond))

	err := service.RecordFromCapture(requestContext, RecordInput{
		ExecutionID: "exec-canceled", ProviderID: providers.OpenCodeGoID, ModelID: "model",
		Capture: capture.Result{FinalEventData: []byte(`{"usage":{"input_tokens":4,"output_tokens":2,"total_tokens":6}}`)},
	})
	if err != nil {
		t.Fatalf("RecordFromCapture: %v", err)
	}
	if !called {
		t.Fatal("repository was not called")
	}
}

func TestRecordFromCaptureReturnsPersistenceTimeout(t *testing.T) {
	started := make(chan struct{})
	service := NewService(repositoryFunc(func(ctx context.Context, _ Record) (bool, error) {
		close(started)
		<-ctx.Done()
		return false, ctx.Err()
	}), WithPersistenceTimeout(20*time.Millisecond))

	start := time.Now()
	err := service.RecordFromCapture(context.Background(), RecordInput{
		ExecutionID: "exec-timeout", ProviderID: providers.OpenCodeGoID, ModelID: "model",
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RecordFromCapture error = %v, want deadline exceeded", err)
	}
	select {
	case <-started:
	default:
		t.Fatal("repository was not called")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("persistence took %v, want bounded completion", elapsed)
	}
}
