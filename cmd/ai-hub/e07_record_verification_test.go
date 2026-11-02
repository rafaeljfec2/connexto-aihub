package main

import (
	"aihub/internal/execution"
	"aihub/internal/providers"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
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

func TestE07RecordVerificationFollowsE05TerminalSemantics(t *testing.T) {
	cases := []struct {
		name   string
		status int
		opener func() (*http.Response, error)
	}{
		{
			name:   "upstream 2xx fully read",
			status: http.StatusOK,
			opener: func() (*http.Response, error) {
				return e07FixedResponse(http.StatusOK, "synthetic success body"), nil
			},
		},
		{
			name:   "upstream 4xx fully read",
			status: http.StatusBadRequest,
			opener: func() (*http.Response, error) {
				return e07FixedResponse(http.StatusBadRequest, `{"error":{"type":"invalid_request_error"}}`), nil
			},
		},
		{
			name:   "upstream 401 fully read",
			status: http.StatusUnauthorized,
			opener: func() (*http.Response, error) {
				return e07FixedResponse(http.StatusUnauthorized, `{"error":{"type":"authentication_error"}}`), nil
			},
		},
		{
			name:   "transport failure",
			status: http.StatusBadGateway,
			opener: func() (*http.Response, error) {
				return nil, errors.New("synthetic transport failure")
			},
		},
		{
			name:   "body read failure",
			status: http.StatusOK,
			opener: func() (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       e07FailingBody{},
				}, nil
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := e07StartDeterministicRecords(t, func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
				return testCase.opener()
			})
			requestID := "corr_semantic_" + strings.ReplaceAll(testCase.name, " ", "_")
			fixture.execute(t, e07ValidationCommand(requestID))

			expectations := e07BuildExpectedRecords(
				[]e07RequestCapture{{ID: requestID, Status: testCase.status}},
				fixture.recorder.Snapshot(),
				nil,
			)
			assertions := e07AssertRecords(context.Background(), fixture.reader, expectations)

			if !assertions.Pass() {
				t.Fatalf("record assertions failed: %s", assertions.Summary("deterministic"))
			}
			if assertions.Matched != 1 || assertions.ByIDMatched != 1 {
				t.Fatalf("record assertions did not correlate the execution: %s", assertions.Summary("deterministic"))
			}
		})
	}
}
