package execution

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStreamInactivityClosesAndJoinsReader(t *testing.T) {
	body := newBlockingBody("first chunk")
	sink := &sinkProbe{wrote: make(chan string, 1)}
	manager := newTestManager(30*time.Millisecond, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		return responseFor(body, http.StatusOK), nil
	})
	result, err := manager.Execute(context.Background(), validCommand(), sink)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if result.Outcome != OutcomeTimedOut || result.Cause != CauseInactivity || !result.Committed {
		t.Fatalf("result = %+v, want committed timed_out(inactivity)", result)
	}
	if got := body.closeCount.Load(); got != 1 {
		t.Fatalf("body close count = %d, want 1", got)
	}
	if !manager.Drained() {
		t.Fatal("manager still has active execution work after inactivity cleanup")
	}
	if _, _, chunks := sink.snapshot(); !reflect.DeepEqual(chunks, []string{"first chunk"}) {
		t.Fatalf("chunks = %q, want the first progressive chunk", chunks)
	}
}

func TestUpstreamReadFailureAndGatewayWriteFailureHaveDistinctResults(t *testing.T) {
	t.Run("upstream read", func(t *testing.T) {
		body := &readErrorBody{}
		manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
			return responseFor(body, http.StatusOK), nil
		})
		result, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
		if err != nil {
			t.Fatalf("Execute() returned error: %v", err)
		}
		if result.Outcome != OutcomeFailed || result.Cause != CauseUpstreamRead {
			t.Fatalf("result = %+v, want failed(upstream_read)", result)
		}
		if body.closed != 1 || !manager.Drained() {
			t.Fatal("upstream failure did not close and drain the execution")
		}
	})

	t.Run("gateway write", func(t *testing.T) {
		body := newBlockingBody("chunk")
		sink := &sinkProbe{writeErr: errors.New("client write failed")}
		manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
			return responseFor(body, http.StatusOK), nil
		})
		result, err := manager.Execute(context.Background(), validCommand(), sink)
		if err != nil {
			t.Fatalf("Execute() returned error: %v", err)
		}
		if result.Outcome != OutcomeCanceled || result.Cause != CauseClient {
			t.Fatalf("result = %+v, want canceled(client)", result)
		}
		if body.closeCount.Load() != 1 || !manager.Drained() {
			t.Fatal("gateway write failure did not close and drain the execution")
		}
	})
}

type readErrorBody struct {
	closed int
}

func (b *readErrorBody) Read([]byte) (int, error) { return 0, errors.New("upstream read failed") }
func (b *readErrorBody) Close() error             { b.closed++; return nil }

func TestStreamRecordsTimeToFirstTokenOnFirstChunk(t *testing.T) {
	manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		return responseFor(io.NopCloser(strings.NewReader("token")), http.StatusOK), nil
	})
	result, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if result.Outcome != OutcomeCompleted {
		t.Fatalf("result = %+v, want completed", result)
	}
	if result.TimeToFirstTokenNS == nil || *result.TimeToFirstTokenNS < 0 {
		t.Fatalf("time_to_first_token_ns = %v, want observed latency", result.TimeToFirstTokenNS)
	}
	var firstOutput *TelemetryEvent
	for index := range result.TelemetryEvents {
		if result.TelemetryEvents[index].Kind == EventFirstOutput {
			firstOutput = &result.TelemetryEvents[index]
			break
		}
	}
	if firstOutput == nil || firstOutput.DurationNS == nil {
		t.Fatal("missing first_output telemetry event")
	}
	if *firstOutput.DurationNS != *result.TimeToFirstTokenNS {
		t.Fatalf("first_output duration = %d, time_to_first_token_ns = %d", *firstOutput.DurationNS, *result.TimeToFirstTokenNS)
	}
}

func TestCompletedEmptyResponseRetainsUpstreamStatus(t *testing.T) {
	manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		return responseFor(io.NopCloser(strings.NewReader("")), http.StatusNoContent), nil
	})
	result, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if result.Outcome != OutcomeCompleted || result.UpstreamStatus != http.StatusNoContent {
		t.Fatalf("result = %+v, want completed upstream 204", result)
	}
}

func TestBodyCloseFailurePreventsCleanDrain(t *testing.T) {
	manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		return responseFor(&closeErrorBody{}, http.StatusOK), nil
	})
	result, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if result.Outcome != OutcomeCompleted {
		t.Fatalf("outcome = %s, want completed upstream exchange", result.Outcome)
	}
	if manager.Drained() {
		t.Fatal("manager reported a clean drain after body Close failed")
	}
	if err := manager.Wait(context.Background()); !errors.Is(err, ErrCleanupUnconfirmed) {
		t.Fatalf("Wait() error = %v, want ErrCleanupUnconfirmed", err)
	}
}

type closeErrorBody struct{}

func (*closeErrorBody) Read([]byte) (int, error) { return 0, io.EOF }
func (*closeErrorBody) Close() error             { return errors.New("close failed") }
