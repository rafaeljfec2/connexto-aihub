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

func TestE07RecordVerificationKeepsAssertionsIndependent(t *testing.T) {
	executionID := strings.Repeat("d", 32)
	requestID := "corr_independent_assertions"
	expectedRecord := e07ExecutionRecord{
		ExecutionID: executionID,
		RequestID:   &requestID,
		Provider:    providers.OpenCodeGoID,
		Outcome:     string(execution.OutcomeCompleted),
		Usage:       []byte("null"),
	}
	expectations := e07RecordExpectations{
		Records: map[string]e07ExpectedRecord{
			executionID: {RequestID: requestID, ExecutionID: executionID, Outcome: string(execution.OutcomeCompleted)},
		},
	}

	cases := []struct {
		name   string
		reader *e07FakeRecordReader
		check  func(*testing.T, e07RecordAssertions)
	}{
		{
			name: "list error does not suppress by-id",
			reader: &e07FakeRecordReader{
				listErr: errors.New("synthetic list failure"),
				records: map[string]e07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07RecordAssertions) {
				if assertions.ListOK {
					t.Fatal("list assertion accepted a failed list query")
				}
				if !assertions.ByIDOK || assertions.ByIDMatched != 1 {
					t.Fatal("by-id assertion was suppressed by the list failure")
				}
			},
		},
		{
			name: "outcome mismatch does not suppress by-id",
			reader: &e07FakeRecordReader{
				listRecords: []e07ExecutionRecord{{
					ExecutionID: executionID,
					RequestID:   &requestID,
					Provider:    providers.OpenCodeGoID,
					Outcome:     string(execution.OutcomeFailed),
					Usage:       []byte("null"),
				}},
				records: map[string]e07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07RecordAssertions) {
				if assertions.OutcomeOK {
					t.Fatal("outcome assertion accepted an E05 outcome mismatch")
				}
				if !assertions.ByIDOK {
					t.Fatal("by-id assertion was suppressed by the outcome mismatch")
				}
			},
		},
		{
			name: "missing record does not suppress by-id",
			reader: &e07FakeRecordReader{
				records: map[string]e07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07RecordAssertions) {
				if assertions.CorrelationOK || assertions.Matched != 0 {
					t.Fatal("correlation assertion accepted a missing record")
				}
				if !assertions.ByIDOK {
					t.Fatal("by-id assertion was suppressed by a missing list row")
				}
			},
		},
		{
			name: "schema mismatch does not suppress outcome or by-id",
			reader: &e07FakeRecordReader{
				listRecords: []e07ExecutionRecord{{
					ExecutionID: executionID,
					RequestID:   &requestID,
					Provider:    providers.OpenCodeGoID,
					Outcome:     string(execution.OutcomeCompleted),
					Usage:       []byte(`{"total_tokens":1}`),
				}},
				records: map[string]e07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07RecordAssertions) {
				if assertions.SchemaOK {
					t.Fatal("schema assertion accepted a non-null usage value")
				}
				if !assertions.OutcomeOK || !assertions.ByIDOK {
					t.Fatal("schema mismatch suppressed another assertion")
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assertions := e07AssertRecords(context.Background(), testCase.reader, expectations)
			testCase.check(t, assertions)
		})
	}
}

func TestE07RecordAssertionsSurviveParentCancellation(t *testing.T) {
	executionID := strings.Repeat("f", 32)
	requestID := "corr_parent_cancellation"
	expectedRecord := e07ExecutionRecord{
		ExecutionID: executionID,
		RequestID:   &requestID,
		Provider:    providers.OpenCodeGoID,
		Outcome:     string(execution.OutcomeCompleted),
		Usage:       []byte("null"),
	}
	parent, cancel := context.WithCancel(context.Background())
	reader := &e07FakeRecordReader{
		listErr: errors.New("synthetic slow list failure"),
		records: map[string]e07ExecutionRecord{executionID: expectedRecord},
		onList:  cancel,
	}
	expectations := e07RecordExpectations{
		Records: map[string]e07ExpectedRecord{
			executionID: {RequestID: requestID, ExecutionID: executionID, Outcome: string(execution.OutcomeCompleted)},
		},
	}

	assertions := e07AssertRecords(parent, reader, expectations)
	if !assertions.ByIDOK {
		t.Fatalf("by-id assertion was suppressed by parent cancellation: %s", assertions.Summary("cancelled_parent"))
	}
	if len(reader.getContextErrs) == 0 {
		t.Fatal("by-id query was not attempted")
	}
	for index, err := range reader.getContextErrs {
		if err != nil {
			t.Fatalf("by-id query %d inherited the cancelled parent: %v", index, err)
		}
	}
}

type e07FakeRecordReader struct {
	listRecords    []e07ExecutionRecord
	listErr        error
	records        map[string]e07ExecutionRecord
	getErr         error
	onList         func()
	contextErrs    []error
	getContextErrs []error
}

func (reader *e07FakeRecordReader) List(ctx context.Context) ([]e07ExecutionRecord, error) {
	reader.contextErrs = append(reader.contextErrs, ctx.Err())
	if reader.onList != nil {
		reader.onList()
	}
	if reader.listErr != nil {
		return nil, reader.listErr
	}
	return append([]e07ExecutionRecord(nil), reader.listRecords...), nil
}

func (reader *e07FakeRecordReader) Get(ctx context.Context, executionID string) (e07ExecutionRecord, error) {
	reader.getContextErrs = append(reader.getContextErrs, ctx.Err())
	if reader.getErr != nil {
		return e07ExecutionRecord{}, reader.getErr
	}
	record, ok := reader.records[executionID]
	if !ok {
		return e07ExecutionRecord{}, errors.New("record not found")
	}
	return record, nil
}

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

func TestE07UpstreamErrorEnvelopeClassification(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{name: "generic error envelope", body: `{"error":{"type":"invalid_request_error","message":"unknown model"}}`, want: true},
		{name: "error key as string", body: `{"error":"upstream failure"}`, want: true},
		{name: "non-error body", body: `{"id":"synthetic","object":"response"}`, want: false},
		{name: "null error", body: `{"error":null}`, want: false},
		{name: "empty body", body: ``, want: false},
		{name: "invalid json", body: `not-json`, want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := e07ErrorEnvelopePresent([]byte(testCase.body)); got != testCase.want {
				t.Fatalf("envelope present = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestE07UpstreamProbeDoesNotRequireCrossCallBodyEquality(t *testing.T) {
	observation := e07UpstreamProbeObservation{
		DirectStatus:    http.StatusBadRequest,
		HubStatus:       http.StatusBadRequest,
		EnvelopePresent: true,
		HubBodyRead:     true,
	}
	if !observation.PassthroughObserved() {
		t.Fatal("a client error with a generic envelope was not accepted")
	}
	if !strings.Contains(observation.Summary(), "status_preserved=true") {
		t.Fatal("upstream diagnostic does not report status preservation separately")
	}

	observation.HubStatus = http.StatusBadGateway
	if observation.PassthroughObserved() {
		t.Fatal("a gateway-generated status was accepted as upstream passthrough")
	}

	observation.HubStatus = http.StatusNotFound
	if observation.PassthroughObserved() {
		t.Fatal("a Hub status that differs from the direct provider status was accepted")
	}

	observation.HubStatus = http.StatusBadRequest
	observation.EnvelopePresent = false
	if observation.PassthroughObserved() {
		t.Fatal("a non-envelope body was accepted as upstream passthrough")
	}
	if strings.Contains(observation.Summary(), "body=") {
		t.Fatal("upstream diagnostic summary exposes body content")
	}
}
