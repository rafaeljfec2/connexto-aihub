package execution_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/providers"
)

func TestExecuteLogsUpstreamErrorSummary(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	const summary = "Missing required parameter: model; code=invalid_prompt"
	open := func(_ context.Context, _ execution.Target, _ []byte, _, _ string) (*http.Response, error) {
		header := make(http.Header)
		header.Set(providers.InternalUpstreamErrorDetailHeader, summary)
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     header,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"upstream_error","message":"sanitized"}}`)),
		}, nil
	}

	manager := execution.NewManager(logger, "openrouter", open, time.Minute)
	sink := &captureSink{}
	result, err := manager.Execute(context.Background(), execution.Command{
		RequestID:         "req-log",
		ExecutionID:       "exec-log",
		ProviderSessionID: "session-log",
		Target:            execution.Target{ProviderID: "openrouter", ModelID: "anthropic/claude-haiku-4.5"},
		Payload:           []byte(`{"model":"anthropic/claude-haiku-4.5","input":"hi"}`),
		ContentType:       "application/json",
	}, sink)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.UpstreamStatus != http.StatusBadRequest {
		t.Fatalf("upstream status = %d", result.UpstreamStatus)
	}
	if result.UpstreamErrorSummary != summary {
		t.Fatalf("result summary = %q", result.UpstreamErrorSummary)
	}
	logText := logBuf.String()
	if !strings.Contains(logText, "upstream_error="+summary) && !strings.Contains(logText, "upstream_error") {
		t.Fatalf("log missing upstream_error field: %s", logText)
	}
}

type captureSink struct{}

func (captureSink) Commit(int, string) {}

func (captureSink) WriteChunk(chunk []byte) error {
	_, err := io.Copy(io.Discard, strings.NewReader(string(chunk)))
	return err
}
