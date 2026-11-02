package support

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
type E07DiscardSink struct{}

func (E07DiscardSink) Commit(int, string) {}

func (E07DiscardSink) WriteChunk([]byte) error { return nil }

// e07FailingBody fails every read, exercising the E05 upstream-read failure
// branch without a network dependency.
type E07FailingBody struct{}

func (E07FailingBody) Read([]byte) (int, error) { return 0, errors.New("synthetic read failure") }

func (E07FailingBody) Close() error { return nil }

func E07FixedResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func E07DiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// e07DeterministicRecords composes the real E05 manager, the real E06 records
// service, and the real E06 query handlers without any provider or Codex
// process.
type E07DeterministicRecords struct {
	Reader   E07RecordReader
	Manager  *execution.Manager
	Recorder *E07TerminalRecorder
	Service  *records.Service
}

func E07StartDeterministicRecords(t *testing.T, opener execution.OpenResponses) *E07DeterministicRecords {
	t.Helper()

	service := records.New(records.Options{
		Dir:               filepath.Join(t.TempDir(), "records"),
		Logger:            E07DiscardLogger(),
		MaxAge:            720 * time.Hour,
		MaxCount:          10000,
		MaxStorageBytes:   64 << 20,
		SegmentBytes:      1 << 20,
		RetentionInterval: time.Hour,
	})
	if err := service.Start(context.Background()); err != nil {
		t.Fatal("deterministic records service could not start")
	}

	recorder := NewE07TerminalRecorder(service)
	manager := execution.NewManager(E07DiscardLogger(), providers.OpenCodeGoID, opener, time.Second,
		execution.WithObserver(recorder))

	mux := http.NewServeMux()
	mux.Handle("GET /v1/records/executions", gateway.NewExecutionRecordsHandler(service))
	mux.Handle("GET /v1/records/executions/{execution_id}", gateway.NewExecutionRecordHandler(service))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	return &E07DeterministicRecords{
		Reader:   &E07HTTPRecordReader{BaseURL: server.URL, GatewayKey: "validation-gateway-key"},
		Manager:  manager,
		Recorder: recorder,
		Service:  service,
	}
}

func (fixture *E07DeterministicRecords) Execute(t *testing.T, command execution.Command) {
	t.Helper()

	if _, err := fixture.Manager.Execute(context.Background(), command, E07DiscardSink{}); err != nil {
		t.Fatalf("deterministic execution failed: %v", err)
	}
	if err := fixture.Service.Stop(context.Background()); err != nil {
		t.Fatalf("deterministic records drain failed: %v", err)
	}
}

func E07ValidationCommand(requestID string) execution.Command {
	return execution.Command{
		RequestID:         requestID,
		Target:            execution.Target{ProviderID: providers.OpenCodeGoID},
		ProviderSessionID: "aihub-ses-validation",
		Payload:           []byte(`{"input":"synthetic validation request"}`),
		ContentType:       "application/json",
	}
}
