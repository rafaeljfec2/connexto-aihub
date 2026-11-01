package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/records"
)

// e07DiscardSink is the minimal execution sink for deterministic helpers.
type e07DiscardSink struct{}

func (e07DiscardSink) Commit(int, string)      {}
func (e07DiscardSink) WriteChunk([]byte) error { return nil }

// e07FailingBody fails every read, exercising the E05 upstream-read failure
// branch without a network dependency.
type e07FailingBody struct{}

func (e07FailingBody) Read([]byte) (int, error) { return 0, errors.New("synthetic read failure") }
func (e07FailingBody) Close() error             { return nil }

func e07FixedResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func e07DiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// e07DeterministicRecords composes the real E05 manager, the real E06 records
// service, and the real E06 query handlers without any provider or Codex
// process.
type e07DeterministicRecords struct {
	reader   e07RecordReader
	manager  *execution.Manager
	recorder *e07TerminalRecorder
	service  *records.Service
}

func e07StartDeterministicRecords(t *testing.T, opener execution.OpenResponses) *e07DeterministicRecords {
	t.Helper()

	service := records.New(records.Options{
		Dir:               filepath.Join(t.TempDir(), "records"),
		Logger:            e07DiscardLogger(),
		MaxAge:            720 * time.Hour,
		MaxCount:          10000,
		MaxStorageBytes:   64 << 20,
		SegmentBytes:      1 << 20,
		RetentionInterval: time.Hour,
	})
	if err := service.Start(context.Background()); err != nil {
		t.Fatal("deterministic records service could not start")
	}

	recorder := newE07TerminalRecorder(service)
	manager := execution.NewManager(e07DiscardLogger(), providers.OpenCodeGoID, opener, time.Second,
		execution.WithObserver(recorder))

	mux := http.NewServeMux()
	mux.Handle("GET /v1/records/executions", gateway.NewExecutionRecordsHandler(service))
	mux.Handle("GET /v1/records/executions/{execution_id}", gateway.NewExecutionRecordHandler(service))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	return &e07DeterministicRecords{
		reader:   &e07HTTPRecordReader{baseURL: server.URL, gatewayKey: "validation-gateway-key"},
		manager:  manager,
		recorder: recorder,
		service:  service,
	}
}

func (fixture *e07DeterministicRecords) execute(t *testing.T, command execution.Command) {
	t.Helper()

	if _, err := fixture.manager.Execute(context.Background(), command, e07DiscardSink{}); err != nil {
		t.Fatalf("deterministic execution failed: %v", err)
	}
	if err := fixture.service.Stop(context.Background()); err != nil {
		t.Fatalf("deterministic records drain failed: %v", err)
	}
}

func e07ValidationCommand(requestID string) execution.Command {
	return execution.Command{
		RequestID:         requestID,
		Target:            execution.Target{ProviderID: providers.OpenCodeGoID},
		ProviderSessionID: "aihub-ses-validation",
		Payload:           []byte(`{"input":"synthetic validation request"}`),
		ContentType:       "application/json",
	}
}

func TestE07ExpectedRecordsDeriveOutcomeFromE05TerminalObservation(t *testing.T) {
	captures := []e07RequestCapture{
		{ID: "corr_request_400", Status: http.StatusBadRequest},
		{ID: "corr_request_200", Status: http.StatusOK},
		{ID: "corr_request_local_failure", Status: http.StatusBadGateway},
		{ID: "corr_request_upstream_401", Status: http.StatusUnauthorized},
		{ID: "corr_authentication_probe", Status: http.StatusUnauthorized},
		{ID: "corr_request_unobserved", Status: http.StatusOK},
	}
	terminals := []e07TerminalObservation{
		{RequestID: "corr_request_400", ExecutionID: strings.Repeat("a", 32), ProviderID: providers.OpenCodeGoID, Outcome: string(execution.OutcomeCompleted)},
		{RequestID: "corr_request_200", ExecutionID: strings.Repeat("b", 32), ProviderID: providers.OpenCodeGoID, Outcome: string(execution.OutcomeCompleted)},
		{RequestID: "corr_request_local_failure", ExecutionID: strings.Repeat("c", 32), ProviderID: providers.OpenCodeGoID, Outcome: string(execution.OutcomeFailed)},
		{RequestID: "corr_request_upstream_401", ExecutionID: strings.Repeat("d", 32), ProviderID: providers.OpenCodeGoID, Outcome: string(execution.OutcomeCompleted)},
	}

	expectations := e07BuildExpectedRecords(captures, terminals, []string{"corr_authentication_probe"})

	if got, want := len(expectations.Records), 4; got != want {
		t.Fatalf("expected records = %d, want %d", got, want)
	}
	if got, want := expectations.UnmatchedCaptures, 1; got != want {
		t.Fatalf("unmatched captures = %d, want %d", got, want)
	}
	if got := expectations.Records[strings.Repeat("a", 32)].Outcome; got != string(execution.OutcomeCompleted) {
		t.Fatalf("upstream 4xx expectation = %q, want the E05 completed outcome", got)
	}
	if got := expectations.Records[strings.Repeat("c", 32)].Outcome; got != string(execution.OutcomeFailed) {
		t.Fatalf("local failure expectation = %q, want the E05 failed outcome", got)
	}
	if got := expectations.Records[strings.Repeat("d", 32)].Outcome; got != string(execution.OutcomeCompleted) {
		t.Fatalf("upstream 401 expectation = %q, want the E05 completed outcome", got)
	}
	for _, record := range expectations.Records {
		if record.RequestID == "corr_authentication_probe" {
			t.Fatal("a request rejected before E05 admission produced an expectation")
		}
		if record.RequestID == "corr_request_unobserved" {
			t.Fatal("a capture without an E05 terminal observation produced an expectation")
		}
	}
}
