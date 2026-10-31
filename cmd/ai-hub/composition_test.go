package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"aihub/internal/app"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/observability"
	"aihub/internal/records"
	"aihub/internal/server"
)

// orderedRecordsModule wraps the real records service and records whether the
// E05 registry was already drained when the records module was stopped.
type orderedRecordsModule struct {
	*records.Service

	manager       *execution.Manager
	stops         int
	drainedAtStop bool
}

func (m *orderedRecordsModule) Stop(ctx context.Context) error {
	m.stops++
	m.drainedAtStop = m.manager.Drained()

	return m.Service.Stop(ctx)
}

type recordsProviderProbe struct {
	records        *records.Service
	recordsModule  *orderedRecordsModule
	stops          int
	sawRecord      bool
	recordsStopped bool
}

func (*recordsProviderProbe) Name() string                { return "provider-probe" }
func (*recordsProviderProbe) Start(context.Context) error { return nil }

func (p *recordsProviderProbe) Stop(context.Context) error {
	p.stops++
	p.recordsStopped = p.recordsModule.stops == 1
	page, err := p.records.List(records.Filter{Limit: 10})
	p.sawRecord = err == nil && len(page.Records) == 1

	return nil
}

type delayedEOFBody struct {
	readStarted chan struct{}
	delay       time.Duration
	once        sync.Once
}

func (b *delayedEOFBody) Read([]byte) (int, error) {
	b.once.Do(func() { close(b.readStarted) })
	time.Sleep(b.delay)

	return 0, io.EOF
}

func (*delayedEOFBody) Close() error { return nil }

func TestApplicationShutdownFlushesRecordsBeforeProviderRelease(t *testing.T) {
	logger := observability.NewLogger(io.Discard, slog.LevelInfo, "text")
	body := &delayedEOFBody{readStarted: make(chan struct{}), delay: 100 * time.Millisecond}

	recordsModule := &orderedRecordsModule{Service: records.New(records.Options{
		Dir:               t.TempDir(),
		Logger:            logger,
		MaxAge:            720 * time.Hour,
		MaxCount:          10000,
		MaxStorageBytes:   64 << 20,
		SegmentBytes:      1 << 20,
		RetentionInterval: time.Hour,
	})}
	manager := execution.NewManager(logger, "opencode-go",
		func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
		}, time.Hour, execution.WithObserver(recordsModule))
	recordsModule.manager = manager

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = manager.Execute(r.Context(), execution.Command{
			Target:            execution.Target{ProviderID: "opencode-go"},
			ProviderSessionID: "session-composition-test",
			Payload:           []byte("{}"),
			ContentType:       "application/json",
		}, shutdownSink{writer: w})
	})

	cfg := config.Default().Server
	cfg.Port = 0
	cfg.ShutdownTimeout = time.Second
	httpServer := server.New(cfg, handler, logger)
	lifecycle := newExecutionLifecycle(httpServer, manager, cfg.ShutdownTimeout)
	provider := &recordsProviderProbe{records: recordsModule.Service, recordsModule: recordsModule}

	application, err := app.New(logger, newGuardedProviderModule(provider, manager), recordsModule, lifecycle)
	if err != nil {
		t.Fatalf("app.New() returned error: %v", err)
	}
	if err := application.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	go func() {
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := application.Stop(ctx); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	if recordsModule.stops != 1 || !recordsModule.drainedAtStop {
		t.Fatalf("records stop = %d, drained at stop = %t; want one stop after the E05 drain",
			recordsModule.stops, recordsModule.drainedAtStop)
	}
	if provider.stops != 1 || !provider.sawRecord || !provider.recordsStopped {
		t.Fatalf("provider stops = %d, saw record = %t, records stopped = %t; want release after the records flush",
			provider.stops, provider.sawRecord, provider.recordsStopped)
	}

	page, err := recordsModule.List(records.Filter{Limit: 10})
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(page.Records) != 1 || page.Records[0].Outcome != records.OutcomeCompleted {
		t.Fatalf("retained records = %+v, want one completed execution", page.Records)
	}
}
