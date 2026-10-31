package records

import (
	"context"
	"errors"
	"os"
	"testing"

	"aihub/internal/execution"
)

func TestServiceReportsUnavailableIndex(t *testing.T) {
	ops := &hookOps{mkdirAllHook: func(string, os.FileMode) error { return errors.New("permission denied") }}
	service := New(testOptions(t.TempDir()))
	service.ops = ops

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start() must not fail startup on storage failure: %v", err)
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	if _, _, err := service.Get(executionID(1)); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get() error = %v, want ErrUnavailable", err)
	}
	if _, err := service.List(Filter{Limit: 10}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("List() error = %v, want ErrUnavailable", err)
	}
	if !service.Degraded() {
		t.Fatal("unavailable storage must report degraded history")
	}

	service.OnExecutionFinished(finishedExecution(executionID(1), at(1), execution.OutcomeCompleted, "", 200, true))
	if snapshot := service.Counters(); snapshot.DroppedClosed != 1 {
		t.Fatalf("dropped closed observations = %d, want 1", snapshot.DroppedClosed)
	}
}
