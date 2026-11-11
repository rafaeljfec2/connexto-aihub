package execution

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

type headerTimeoutError struct{}

func (headerTimeoutError) Error() string                 { return "header timeout" }
func (headerTimeoutError) IsResponseHeaderTimeout() bool { return true }

func TestExecutePassesTargetAndPayloadAndPreservesUpstreamStatus(t *testing.T) {
	command := validCommand()
	command.Target.ModelID = "model-from-caller"
	command.Payload = []byte{'{', 0, '\n', 0xff, '}'}
	var gotTarget Target
	var gotPayload []byte
	manager := newTestManager(time.Second, func(_ context.Context, target Target, payload []byte, _, _ string) (*http.Response, error) {
		gotTarget = target
		gotPayload = append([]byte(nil), payload...)
		return responseFor(newEOFBody("provider response"), http.StatusTooManyRequests), nil
	})
	sink := &sinkProbe{}

	result, err := manager.Execute(context.Background(), command, sink)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if gotTarget != command.Target {
		t.Fatalf("target = %+v, want %+v", gotTarget, command.Target)
	}
	if !reflect.DeepEqual(gotPayload, command.Payload) {
		t.Fatalf("payload = %v, want byte-identical %v", gotPayload, command.Payload)
	}
	if result.Outcome != OutcomeCompleted || result.UpstreamStatus != http.StatusTooManyRequests {
		t.Fatalf("result = %+v, want completed upstream 429", result)
	}
	status, contentType, chunks := sink.snapshot()
	if status != http.StatusTooManyRequests || contentType != "text/event-stream" {
		t.Fatalf("sink metadata = %d %q, want upstream metadata", status, contentType)
	}
	if !reflect.DeepEqual(chunks, []string{"provider response"}) {
		t.Fatalf("chunks = %q, want provider body", chunks)
	}
}

func TestExecuteRejectsInvalidTargetBeforeProviderCall(t *testing.T) {
	var calls int
	manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		calls++
		return responseFor(newEOFBody(""), http.StatusOK), nil
	})
	command := validCommand()
	command.Target.ProviderID = "unknown"

	if _, err := manager.Execute(context.Background(), command, &sinkProbe{}); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("Execute() error = %v, want ErrInvalidCommand", err)
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want none", calls)
	}
}

func TestExecuteClassifiesOpeningTimeoutAndTransportFailure(t *testing.T) {
	tests := []struct {
		name    string
		openErr error
		outcome Outcome
		cause   Cause
	}{
		{name: "header timeout", openErr: headerTimeoutError{}, outcome: OutcomeTimedOut, cause: CauseOpening},
		{name: "transport failure", openErr: errors.New("dial failed"), outcome: OutcomeFailed, cause: CauseTransport},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
				return nil, test.openErr
			})
			result, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
			if err != nil {
				t.Fatalf("Execute() returned error: %v", err)
			}
			if result.Outcome != test.outcome || result.Phase != PhaseOpening || result.Cause != test.cause {
				t.Fatalf("result = %+v, want %s/%s", result, test.outcome, test.cause)
			}
		})
	}
}

func TestBeginDrainRejectsNewCommandsWithoutRegistration(t *testing.T) {
	var calls int
	manager := newTestManager(time.Second, func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		calls++
		return responseFor(newEOFBody(""), http.StatusOK), nil
	})
	manager.BeginDrain()

	if _, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Execute() error = %v, want ErrUnavailable", err)
	}
	if calls != 0 || !manager.Drained() {
		t.Fatalf("calls = %d, drained = %t; want no provider call and clean empty registry", calls, manager.Drained())
	}
}

func TestLifecycleLogsOmitPayloadAndGeneratedContent(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	manager := NewManager(logger, "opencode-go", func(context.Context, Target, []byte, string, string) (*http.Response, error) {
		return responseFor(newEOFBody("generated-secret-content"), http.StatusOK), nil
	}, time.Second)
	command := validCommand()
	command.Payload = []byte("request-secret-content")
	if _, err := manager.Execute(context.Background(), command, &sinkProbe{}); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	for _, secret := range []string{"request-secret-content", "generated-secret-content"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("lifecycle logs contain sensitive content %q", secret)
		}
	}
}
