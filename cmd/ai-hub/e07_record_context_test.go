package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/providers"
)

func TestE07RecordQueryContextClampsToPollWindow(t *testing.T) {
	expiredDeadline := time.Now().Add(-time.Second)
	expiredContext, cancelExpired := e07RecordQueryContext(context.Background(), expiredDeadline)
	defer cancelExpired()
	deadline, ok := expiredContext.Deadline()
	if !ok {
		t.Fatal("clamped query context is missing a deadline")
	}
	if time.Until(deadline) > 100*time.Millisecond {
		t.Fatalf("expired poll window produced a %s query budget", time.Until(deadline))
	}
	select {
	case <-expiredContext.Done():
	case <-time.After(time.Second):
		t.Fatal("expired poll window did not cancel the query context")
	}

	futureDeadline := time.Now().Add(5 * time.Second)
	futureContext, cancelFuture := e07RecordQueryContext(context.Background(), futureDeadline)
	defer cancelFuture()
	deadline, ok = futureContext.Deadline()
	if !ok {
		t.Fatal("clamped query context is missing a deadline")
	}
	remaining := time.Until(deadline)
	if remaining > 5*time.Second || remaining < 4*time.Second {
		t.Fatalf("query budget = %s, want it clamped to the remaining poll window", remaining)
	}
}

func TestE07RestartRecordVerificationUsesFreshContext(t *testing.T) {
	executionID := strings.Repeat("e", 32)
	requestID := "corr_restart_fresh_context"
	expectedRecord := e07ExecutionRecord{
		ExecutionID: executionID,
		RequestID:   &requestID,
		Provider:    providers.OpenCodeGoID,
		Outcome:     string(execution.OutcomeCompleted),
		Usage:       []byte("null"),
	}
	reader := &e07FakeRecordReader{
		listRecords: []e07ExecutionRecord{expectedRecord},
		records:     map[string]e07ExecutionRecord{executionID: expectedRecord},
	}
	expectations := e07RecordExpectations{
		Records: map[string]e07ExpectedRecord{
			executionID: {RequestID: requestID, ExecutionID: executionID, Outcome: string(execution.OutcomeCompleted)},
		},
	}
	expired, cancel := context.WithCancel(context.Background())
	cancel()

	assertions := e07AssertRecordsAfterRestart(expired, reader, expectations, time.Second)
	if !assertions.Pass() {
		t.Fatalf("restart assertions failed with a fresh context: %s", assertions.Summary("post_restart"))
	}
	for index, err := range append(reader.contextErrs, reader.getContextErrs...) {
		if err != nil {
			t.Fatalf("restart query %d reused an expired context: %v", index, err)
		}
	}
}
