package execution_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
)

type routedSink struct {
	status int
	body   strings.Builder
}

func (s *routedSink) Commit(status int, _ string) { s.status = status }
func (s *routedSink) WriteChunk(chunk []byte) error {
	_, err := s.body.Write(chunk)
	return err
}

func TestMultiProviderManagerAcceptsExplicitProviderTarget(t *testing.T) {
	var got execution.Target
	manager := execution.NewMultiProviderManager(slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(_ context.Context, target execution.Target, _ []byte, _, _ string) (*http.Response, error) {
			got = target
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: [DONE]\n\n"))}, nil
		}, time.Minute)
	sink := &routedSink{}
	result, err := manager.Execute(context.Background(), execution.Command{
		Target:            execution.Target{ProviderID: "openrouter", ModelID: "anthropic/claude-sonnet-4.5"},
		ProviderSessionID: "session-1",
		Payload:           []byte(`{"model":"anthropic/claude-sonnet-4.5"}`),
		ContentType:       "application/json",
	}, sink)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.ProviderID != "openrouter" || got.ModelID != "anthropic/claude-sonnet-4.5" {
		t.Fatalf("opener target = %+v", got)
	}
	if result.Outcome != execution.OutcomeCompleted || sink.status != http.StatusOK || !strings.Contains(sink.body.String(), "[DONE]") {
		t.Fatalf("result/sink = %+v/%d %q", result, sink.status, sink.body.String())
	}
}
