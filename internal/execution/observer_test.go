package execution

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type observerProbe struct {
	mu            sync.Mutex
	observations  []FinishedExecution
	drainedInCall []bool
	manager       *Manager
	panicOnCall   bool
}

func (p *observerProbe) OnExecutionFinished(finished FinishedExecution) {
	if p.panicOnCall {
		panic("observer failure")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.observations = append(p.observations, finished)
	if p.manager != nil {
		p.drainedInCall = append(p.drainedInCall, p.manager.Drained())
	}
}

func (p *observerProbe) snapshot() ([]FinishedExecution, []bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]FinishedExecution(nil), p.observations...), append([]bool(nil), p.drainedInCall...)
}

func TestManagerNotifiesObserverOnceBeforeDeregistration(t *testing.T) {
	probe := &observerProbe{}
	manager := NewManager(nil, "opencode-go",
		func(context.Context, Target, []byte, string, string) (*http.Response, error) {
			return responseFor(newEOFBody("provider response"), http.StatusOK), nil
		}, time.Second, WithObserver(probe))
	probe.manager = manager

	command := validCommand()
	command.Target.ModelID = "vendor/model"
	result, err := manager.Execute(context.Background(), command, &sinkProbe{})
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	observations, drained := probe.snapshot()
	if len(observations) != 1 {
		t.Fatalf("observer calls = %d, want exactly one", len(observations))
	}

	finished := observations[0]
	if finished.Result.ExecutionID != result.ExecutionID ||
		finished.Result.Outcome != result.Outcome ||
		finished.Result.Phase != result.Phase ||
		finished.Result.Cause != result.Cause ||
		finished.Result.UpstreamStatus != result.UpstreamStatus ||
		finished.Result.Committed != result.Committed ||
		finished.Result.Duration != result.Duration {
		t.Fatalf("observed result = %+v, want %+v", finished.Result, result)
	}
	if finished.RequestID != command.RequestID {
		t.Fatalf("observed request id = %q, want %q", finished.RequestID, command.RequestID)
	}
	if finished.Target != command.Target {
		t.Fatalf("observed target = %+v, want %+v", finished.Target, command.Target)
	}
	if finished.StartedAt.IsZero() || finished.StartedAt.Location() != time.UTC {
		t.Fatalf("observed started_at = %v, want a UTC wall clock", finished.StartedAt)
	}
	if finished.StartedAt.After(time.Now()) {
		t.Fatalf("observed started_at = %v is in the future", finished.StartedAt)
	}
	if len(drained) != 1 || drained[0] {
		t.Fatal("observer must run before the execution leaves the active registry")
	}
	if !manager.Drained() {
		t.Fatal("manager must be drained after Execute returns")
	}
}

func TestManagerIsolatesObserverPanic(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	probe := &observerProbe{panicOnCall: true}
	manager := NewManager(logger, "opencode-go",
		func(context.Context, Target, []byte, string, string) (*http.Response, error) {
			return responseFor(newEOFBody("provider response"), http.StatusOK), nil
		}, time.Second, WithObserver(probe))

	result, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
	if err != nil {
		t.Fatalf("Execute() returned error after observer panic: %v", err)
	}
	if result.Outcome != OutcomeCompleted {
		t.Fatalf("result = %+v, want the unchanged completed result", result)
	}
	if !manager.Drained() {
		t.Fatal("a panicking observer must not keep the execution registered")
	}

	if !strings.Contains(logs.String(), "observer") {
		t.Fatalf("observer failure was not logged: %s", logs.String())
	}
	if strings.Contains(logs.String(), "opaque payload") {
		t.Fatal("observer failure log leaked the request payload")
	}
}

func TestManagerSkipsObserverForRejectedCommands(t *testing.T) {
	probe := &observerProbe{}
	manager := NewManager(nil, "opencode-go",
		func(context.Context, Target, []byte, string, string) (*http.Response, error) {
			return responseFor(newEOFBody(""), http.StatusOK), nil
		}, time.Second, WithObserver(probe))

	command := validCommand()
	command.Target.ProviderID = "unknown"
	if _, err := manager.Execute(context.Background(), command, &sinkProbe{}); err == nil {
		t.Fatal("Execute() must reject the invalid target")
	}

	if observations, _ := probe.snapshot(); len(observations) != 0 {
		t.Fatalf("observer calls = %d, want none for a rejected command", len(observations))
	}
}

func TestManagerNotifiesObserverOnceUnderTerminalRace(t *testing.T) {
	body := newBlockingBody("")
	probe := &observerProbe{}
	manager := NewManager(nil, "opencode-go",
		func(context.Context, Target, []byte, string, string) (*http.Response, error) {
			return responseFor(body, http.StatusOK), nil
		}, time.Second, WithObserver(probe))

	done := make(chan Result, 1)
	go func() {
		result, _ := manager.Execute(context.Background(), validCommand(), &sinkProbe{})
		done <- result
	}()

	awaitSignal(t, body.readStarted, "upstream read")
	manager.CancelActive()

	select {
	case result := <-done:
		if result.Outcome != OutcomeCanceled {
			t.Fatalf("result = %+v, want canceled", result)
		}
	case <-time.After(time.Second):
		t.Fatal("execution did not finish after cancellation")
	}

	if observations, _ := probe.snapshot(); len(observations) != 1 {
		t.Fatalf("observer calls = %d, want exactly one", len(observations))
	}
}

func TestManagerAcceptsNilObserver(t *testing.T) {
	manager := NewManager(nil, "opencode-go",
		func(context.Context, Target, []byte, string, string) (*http.Response, error) {
			return responseFor(newEOFBody("provider response"), http.StatusOK), nil
		}, time.Second, WithObserver(nil))

	if _, err := manager.Execute(context.Background(), validCommand(), &sinkProbe{}); err != nil {
		t.Fatalf("Execute() returned error with a nil observer: %v", err)
	}
}
