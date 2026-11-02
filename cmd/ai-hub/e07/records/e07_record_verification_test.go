package records

import (
	e07support "aihub/cmd/ai-hub/e07/support"
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
	expiredContext, cancelExpired := e07support.E07RecordQueryContext(context.Background(), expiredDeadline)
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
	futureContext, cancelFuture := e07support.E07RecordQueryContext(context.Background(), futureDeadline)
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
	expectedRecord := e07support.E07ExecutionRecord{
		ExecutionID: executionID,
		RequestID:   &requestID,
		Provider:    providers.OpenCodeGoID,
		Outcome:     string(execution.OutcomeCompleted),
		Usage:       []byte("null"),
	}
	reader := &e07support.E07FakeRecordReader{
		ListRecords: []e07support.E07ExecutionRecord{expectedRecord},
		Records:     map[string]e07support.E07ExecutionRecord{executionID: expectedRecord},
	}
	expectations := e07support.E07RecordExpectations{
		Records: map[string]e07support.E07ExpectedRecord{
			executionID: {RequestID: requestID, ExecutionID: executionID, Outcome: string(execution.OutcomeCompleted)},
		},
	}
	expired, cancel := context.WithCancel(context.Background())
	cancel()

	assertions := e07support.E07AssertRecordsAfterRestart(expired, reader, expectations, time.Second)
	if !assertions.Pass() {
		t.Fatalf("restart assertions failed with a fresh context: %s", assertions.Summary("post_restart"))
	}
	for index, err := range append(reader.ContextErrs, reader.GetContextErrs...) {
		if err != nil {
			t.Fatalf("restart query %d reused an expired context: %v", index, err)
		}
	}
}

func TestE07RecordVerificationFollowsE05TerminalSemantics(t *testing.T) {
	cases := []struct {
		name   string
		Status int
		opener func() (*http.Response, error)
	}{
		{
			name:   "upstream 2xx fully read",
			Status: http.StatusOK,
			opener: func() (*http.Response, error) {
				return e07support.E07FixedResponse(http.StatusOK, "synthetic success body"), nil
			},
		},
		{
			name:   "upstream 4xx fully read",
			Status: http.StatusBadRequest,
			opener: func() (*http.Response, error) {
				return e07support.E07FixedResponse(http.StatusBadRequest, `{"error":{"type":"invalid_request_error"}}`), nil
			},
		},
		{
			name:   "upstream 401 fully read",
			Status: http.StatusUnauthorized,
			opener: func() (*http.Response, error) {
				return e07support.E07FixedResponse(http.StatusUnauthorized, `{"error":{"type":"authentication_error"}}`), nil
			},
		},
		{
			name:   "transport failure",
			Status: http.StatusBadGateway,
			opener: func() (*http.Response, error) {
				return nil, errors.New("synthetic transport failure")
			},
		},
		{
			name:   "body read failure",
			Status: http.StatusOK,
			opener: func() (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       e07support.E07FailingBody{},
				}, nil
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := e07support.E07StartDeterministicRecords(t, func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
				return testCase.opener()
			})
			requestID := "corr_semantic_" + strings.ReplaceAll(testCase.name, " ", "_")
			fixture.Execute(t, e07support.E07ValidationCommand(requestID))

			expectations := e07support.E07BuildExpectedRecords(
				[]e07support.E07RequestCapture{{ID: requestID, Status: testCase.Status}},
				fixture.Recorder.Snapshot(),
				nil,
			)
			assertions := e07support.E07AssertRecords(context.Background(), fixture.Reader, expectations)

			if !assertions.Pass() {
				t.Fatalf("record assertions failed: %s", assertions.Summary("deterministic"))
			}
			if assertions.Matched != 1 || assertions.ByIDMatched != 1 {
				t.Fatalf("record assertions did not correlate the execution: %s", assertions.Summary("deterministic"))
			}
		})
	}
}
