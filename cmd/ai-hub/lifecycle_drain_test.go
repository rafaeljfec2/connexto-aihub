package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/observability"
	"aihub/internal/server"
)

func TestExecutionLifecycleDrainsRegisteredWorkBeforeProviderStop(t *testing.T) {
	body := newDrainBody()
	var providerCalls atomic.Int32
	manager := execution.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)), "opencode-go",
		func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
			if providerCalls.Add(1) == 1 {
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
		}, time.Hour)

	executionDone := make(chan execution.Result, 1)
	go func() {
		result, _ := manager.Execute(context.Background(), drainCommand(), &shutdownSink{writer: httptest.NewRecorder()})
		executionDone <- result
	}()
	select {
	case <-body.readStarted:
	case <-time.After(time.Second):
		t.Fatal("registered execution did not start reading the upstream body")
	}

	cfg := config.Default().Server
	cfg.Port = 0
	httpServer := server.New(cfg, http.NotFoundHandler(), observability.NewLogger(io.Discard, slog.LevelInfo, "text"))
	lifecycle := newExecutionLifecycle(httpServer, manager, time.Second)
	if lifecycle.Name() != "http-execution-lifecycle" {
		t.Fatalf("lifecycle name = %q, want http-execution-lifecycle", lifecycle.Name())
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stopDone := make(chan error, 1)
	go func() { stopDone <- lifecycle.Stop(stopCtx) }()
	awaitDrainGate(t, manager)
	close(body.release)

	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("Stop() returned error while registered work drained: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("lifecycle did not finish after registered work completed")
	}
	select {
	case result := <-executionDone:
		if result.Outcome != execution.OutcomeCompleted {
			t.Fatalf("result = %+v, want completed", result)
		}
	case <-time.After(time.Second):
		t.Fatal("execution result was not published after graceful drain")
	}
	if body.closeCount.Load() != 1 || !manager.Drained() {
		t.Fatalf("body closes = %d, manager drained = %t", body.closeCount.Load(), manager.Drained())
	}

	provider := &stopProbe{}
	guard := newGuardedProviderModule(provider, manager)
	if guard.Name() != "provider-probe" {
		t.Fatalf("guarded provider name = %q, want provider-probe", guard.Name())
	}
	if err := guard.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	if err := guard.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() returned error after confirmed drain: %v", err)
	}
	if provider.stops != 1 {
		t.Fatalf("provider Stop() calls = %d, want 1", provider.stops)
	}
}

func awaitDrainGate(t *testing.T, manager *execution.Manager) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		_, err := manager.Execute(context.Background(), drainCommand(), &shutdownSink{writer: httptest.NewRecorder()})
		if errors.Is(err, execution.ErrUnavailable) {
			return
		}
		if err != nil {
			t.Fatalf("probe Execute() returned unexpected error: %v", err)
		}
		select {
		case <-deadline.C:
			t.Fatal("lifecycle did not close the execution admission gate")
		default:
		}
	}
}

func drainCommand() execution.Command {
	return execution.Command{
		Target:            execution.Target{ProviderID: "opencode-go"},
		ProviderSessionID: "session-graceful-drain-test",
		ContentType:       "application/json",
	}
}

type drainBody struct {
	readStarted chan struct{}
	release     chan struct{}
	closeCount  atomic.Int32
}

func newDrainBody() *drainBody {
	return &drainBody{readStarted: make(chan struct{}), release: make(chan struct{})}
}

func (b *drainBody) Read([]byte) (int, error) {
	select {
	case <-b.readStarted:
	default:
		close(b.readStarted)
	}
	<-b.release
	return 0, io.EOF
}

func (b *drainBody) Close() error {
	b.closeCount.Add(1)
	return nil
}
