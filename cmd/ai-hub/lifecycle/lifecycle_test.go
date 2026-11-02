package lifecycle

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aihub/internal/app"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/observability"
	"aihub/internal/server"
)

func TestExecutionLifecycleCancelsAndJoinsBeforeReturning(t *testing.T) {
	body := newShutdownBody()
	manager := execution.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)), "opencode-go",
		func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
		}, time.Hour)
	resultCh := make(chan execution.Result, 1)
	handlerDone := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)
		result, _ := manager.Execute(r.Context(), execution.Command{
			Target:            execution.Target{ProviderID: "opencode-go"},
			ProviderSessionID: "session-shutdown-test",
			Payload:           []byte("{}"),
			ContentType:       "application/json",
		}, shutdownSink{writer: w})
		resultCh <- result
	})
	cfg := config.Default().Server
	cfg.Port = 0
	cfg.ShutdownTimeout = 40 * time.Millisecond
	httpServer := server.New(cfg, handler, observability.NewLogger(io.Discard, slog.LevelInfo, "text"))
	lifecycle := NewExecutionLifecycle(httpServer, manager, cfg.ShutdownTimeout)
	if err := lifecycle.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	t.Cleanup(func() { _ = httpServer.ForceClose() })

	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, err := http.Post("http://"+httpServer.Addr(), "application/json", strings.NewReader("{}"))
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()
	select {
	case <-body.readStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream reader did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := lifecycle.Stop(ctx); err == nil {
		t.Fatal("Stop() returned nil after graceful deadline expired")
	}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("HTTP handler remained active after forced shutdown")
	}
	select {
	case result := <-resultCh:
		if result.Outcome != execution.OutcomeCanceled || result.Cause != execution.CauseShutdown {
			t.Fatalf("result = %+v, want canceled(shutdown)", result)
		}
	case <-time.After(time.Second):
		t.Fatal("execution result was not published after reader join")
	}
	if body.closeCount.Load() != 1 || !manager.Drained() {
		t.Fatalf("body closes = %d, manager drained = %t", body.closeCount.Load(), manager.Drained())
	}
	<-requestDone
}

func TestProviderStopGuardRefusesUnconfirmedCleanup(t *testing.T) {
	body := newStubbornShutdownBody()
	manager := execution.NewManager(nil, "opencode-go",
		func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
		}, time.Hour)
	resultDone := make(chan struct{})
	go func() {
		defer close(resultDone)
		_, _ = manager.Execute(context.Background(), execution.Command{
			Target:            execution.Target{ProviderID: "opencode-go"},
			ProviderSessionID: "session-guard-test",
			ContentType:       "application/json",
		}, shutdownSink{writer: httptest.NewRecorder()})
	}()
	select {
	case <-body.readStarted:
	case <-time.After(time.Second):
		t.Fatal("upstream reader did not start")
	}
	manager.BeginDrain()
	manager.CancelActive()

	probe := &stopProbe{}
	guard := NewGuardedProviderModule(probe, manager)
	if err := guard.Stop(context.Background()); err == nil {
		t.Fatal("provider stop succeeded before execution cleanup was confirmed")
	}
	if probe.stops != 0 {
		t.Fatalf("provider Stop() calls = %d, want zero", probe.stops)
	}

	close(body.release)
	select {
	case <-resultDone:
	case <-time.After(time.Second):
		t.Fatal("execution did not finish after controlled reader release")
	}
	if !manager.Drained() {
		t.Fatal("execution cleanup did not finish after reader release")
	}
}

type shutdownSink struct {
	writer http.ResponseWriter
}

func (s shutdownSink) Commit(status int, contentType string) {
	s.writer.Header().Set("Content-Type", contentType)
	s.writer.WriteHeader(status)
}

func (s shutdownSink) WriteChunk(chunk []byte) error {
	_, err := s.writer.Write(chunk)
	return err
}

type shutdownBody struct {
	readStarted chan struct{}
	closed      chan struct{}
	readOnce    sync.Once
	closeOnce   sync.Once
	closeCount  atomic.Int32
}

func newShutdownBody() *shutdownBody {
	return &shutdownBody{readStarted: make(chan struct{}), closed: make(chan struct{})}
}

func (b *shutdownBody) Read([]byte) (int, error) {
	b.readOnce.Do(func() { close(b.readStarted) })
	<-b.closed
	return 0, errors.New("body closed")
}

func (b *shutdownBody) Close() error {
	b.closeCount.Add(1)
	b.closeOnce.Do(func() { close(b.closed) })
	return nil
}

type stubbornShutdownBody struct {
	readStarted chan struct{}
	release     chan struct{}
	readOnce    sync.Once
}

func newStubbornShutdownBody() *stubbornShutdownBody {
	return &stubbornShutdownBody{readStarted: make(chan struct{}), release: make(chan struct{})}
}

func (b *stubbornShutdownBody) Read([]byte) (int, error) {
	b.readOnce.Do(func() { close(b.readStarted) })
	<-b.release
	return 0, io.EOF
}

func (*stubbornShutdownBody) Close() error { return nil }

type stopProbe struct {
	stops int
}

func (*stopProbe) Name() string                 { return "provider-probe" }
func (*stopProbe) Start(context.Context) error  { return nil }
func (p *stopProbe) Stop(context.Context) error { p.stops++; return nil }

var _ app.Module = (*stopProbe)(nil)
