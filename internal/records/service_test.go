package records

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
)

func testOptions(dir string) Options {
	return Options{
		Dir:               dir,
		Logger:            testLogger(),
		MaxAge:            720 * time.Hour,
		MaxCount:          10000,
		MaxStorageBytes:   64 << 20,
		SegmentBytes:      1 << 20,
		RetentionInterval: time.Hour,
	}
}

func startTestService(t *testing.T, opts Options) *Service {
	t.Helper()

	service := New(opts)
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	return service
}

func finishedExecution(id string, completedAt time.Time, outcome execution.Outcome, cause execution.Cause, status int, committed bool) execution.FinishedExecution {
	duration := 1500 * time.Millisecond

	return execution.FinishedExecution{
		Result: execution.Result{
			ExecutionID:    id,
			Outcome:        outcome,
			Phase:          execution.PhaseForwarding,
			Cause:          cause,
			UpstreamStatus: status,
			Committed:      committed,
			Duration:       duration,
		},
		RequestID: "req-0123456789abcdef",
		Target:    execution.Target{ProviderID: "opencode-go"},
		StartedAt: completedAt.Add(-duration).UTC(),
	}
}

func TestServiceRecordsFinishedExecutions(t *testing.T) {
	dir := t.TempDir()
	service := startTestService(t, testOptions(dir))

	finished := finishedExecution(executionID(1), at(1), execution.OutcomeCompleted, "", 200, true)
	service.OnExecutionFinished(finished)

	eventually(t, time.Second, func() bool {
		record, ok, err := service.Get(executionID(1))
		return err == nil && ok && record.Outcome == OutcomeCompleted
	})

	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	reopened := startTestService(t, testOptions(dir))
	record, ok, err := reopened.Get(executionID(1))
	if err != nil || !ok {
		t.Fatalf("recovered record = %+v, found = %t, error = %v", record, ok, err)
	}
	if record.UpstreamStatus == nil || *record.UpstreamStatus != 200 {
		t.Fatalf("upstream status = %v, want 200", record.UpstreamStatus)
	}
	if record.CompletedAt.Sub(record.StartedAt) != 1500*time.Millisecond {
		t.Fatalf("completed_at - started_at = %v, want the recorded duration", record.CompletedAt.Sub(record.StartedAt))
	}
}

func TestServiceRejectsInvalidAndDuplicateObservations(t *testing.T) {
	service := startTestService(t, testOptions(t.TempDir()))

	invalid := finishedExecution(executionID(1), at(1), execution.OutcomeCompleted, "", 200, true)
	invalid.Target.ProviderID = ""
	service.OnExecutionFinished(invalid)

	service.OnExecutionFinished(finishedExecution(executionID(2), at(2), execution.OutcomeCompleted, "", 200, true))
	service.OnExecutionFinished(finishedExecution(executionID(2), at(2), execution.OutcomeCompleted, "", 200, true))

	eventually(t, time.Second, func() bool {
		snapshot := service.Counters()
		return snapshot.InvalidRecord == 1 && snapshot.DuplicateObservation == 1
	})
	eventually(t, time.Second, func() bool {
		page, err := service.List(Filter{Limit: 10})
		return err == nil && len(page.Records) == 1
	})

	page, err := service.List(Filter{Limit: 10})
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(page.Records) != 1 {
		t.Fatalf("records = %d, want only the first accepted value", len(page.Records))
	}
}

func TestServiceQueueOverflowDropsNewestAndDegrades(t *testing.T) {
	block := make(chan struct{})
	entered := make(chan struct{}, 1)
	ops := &hookOps{}
	wrapAppend(ops, blockingWriteHook(block, entered))

	opts := testOptions(t.TempDir())
	service := New(opts)
	service.ops = ops
	service.flushTimeout = 10 * time.Second
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	service.OnExecutionFinished(finishedExecution(executionID(1), at(1), execution.OutcomeCompleted, "", 200, true))
	<-entered

	for n := 2; n <= queueCapacity+1; n++ {
		service.OnExecutionFinished(finishedExecution(executionID(n), at(n), execution.OutcomeCompleted, "", 200, true))
	}

	service.OnExecutionFinished(finishedExecution(executionID(queueCapacity+2), at(2), execution.OutcomeCompleted, "", 200, true))

	snapshot := service.Counters()
	if snapshot.QueueOverflow != 1 {
		t.Fatalf("queue overflow counter = %d, want 1", snapshot.QueueOverflow)
	}
	if !service.Degraded() {
		t.Fatal("queue overflow must mark history degraded")
	}

	close(block)
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
}

func TestServiceStopFlushesAcceptedRecordsInOrder(t *testing.T) {
	dir := t.TempDir()
	service := startTestService(t, testOptions(dir))

	for _, n := range []int{3, 1, 2} {
		service.OnExecutionFinished(finishedExecution(executionID(n), at(n), execution.OutcomeCompleted, "", 200, true))
	}

	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, segmentFileName(1)))
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}

	lines := splitLines(data)
	if len(lines) != 3 {
		t.Fatalf("stored lines = %d, want 3", len(lines))
	}

	want := []string{executionID(3), executionID(1), executionID(2)}
	for position, line := range lines {
		record, decodeErr := decodeRecord([]byte(line))
		if decodeErr != nil {
			t.Fatalf("decode line %d: %v", position, decodeErr)
		}
		if record.ExecutionID != want[position] {
			t.Fatalf("line %d = %s, want %s (FIFO admission order)", position, record.ExecutionID, want[position])
		}
	}

	if service.store == nil || service.store.active != nil {
		t.Fatal("Stop() must close the active segment")
	}
}

func TestServiceStopTimesOutWithoutFailingShutdown(t *testing.T) {
	block := make(chan struct{})
	entered := make(chan struct{}, 1)
	ops := &hookOps{}
	wrapAppend(ops, blockingWriteHook(block, entered))

	var logs bytes.Buffer
	service := New(testOptions(t.TempDir()))
	service.logger = slog.New(slog.NewTextHandler(&logs, nil))
	service.ops = ops
	service.flushTimeout = 40 * time.Millisecond
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	service.OnExecutionFinished(finishedExecution(executionID(1), at(1), execution.OutcomeCompleted, "", 200, true))
	<-entered

	if _, ok, err := service.Get(executionID(1)); err != nil || ok {
		t.Fatalf("record visible before a complete append: found=%t error=%v", ok, err)
	}

	started := time.Now()
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() must not fail shutdown on flush expiry: %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Stop() took %v, want the bounded flush window", elapsed)
	}

	snapshot := service.Counters()
	if snapshot.FlushTimeout == 0 {
		t.Fatal("flush timeout was not counted")
	}
	if snapshot.LostAtShutdown == 0 {
		t.Fatal("records still pending at shutdown were not counted")
	}
	if !strings.Contains(logs.String(), "flush_timeout") {
		t.Fatalf("flush expiry was not logged: %s", logs.String())
	}

	close(block)
	select {
	case <-service.done:
	case <-time.After(time.Second):
		t.Fatal("writer did not finish after the blocked write was released")
	}
}

func TestServiceRejectsObservationsAfterStop(t *testing.T) {
	service := startTestService(t, testOptions(t.TempDir()))
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("first Stop() returned error: %v", err)
	}
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop() returned error: %v", err)
	}

	service.OnExecutionFinished(finishedExecution(executionID(1), at(1), execution.OutcomeCompleted, "", 200, true))

	if snapshot := service.Counters(); snapshot.DroppedClosed == 0 {
		t.Fatal("observation after close must be counted as dropped")
	}
}

func TestServiceQueriesStayConsistentDuringWrites(t *testing.T) {
	service := startTestService(t, testOptions(t.TempDir()))
	done := make(chan struct{})

	go func() {
		defer close(done)
		for n := 1; n <= 50; n++ {
			service.OnExecutionFinished(finishedExecution(executionID(n), at(n), execution.OutcomeCompleted, "", 200, true))
		}
	}()

	for n := 0; n < 100; n++ {
		_, _, _ = service.Get(executionID(1))
		_, _ = service.List(Filter{Limit: 200})
	}

	<-done
	eventually(t, 2*time.Second, func() bool {
		page, err := service.List(Filter{Limit: 200})
		return err == nil && len(page.Records) == 50
	})
}
