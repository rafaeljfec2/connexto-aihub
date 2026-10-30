package app_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"aihub/internal/app"
	"aihub/internal/config"
	"aihub/internal/health"
	"aihub/internal/observability"
	"aihub/internal/server"
)

type fakeModule struct {
	name     string
	events   *[]string
	startErr error
	stopErr  error
}

func (m *fakeModule) Name() string {
	return m.name
}

func (m *fakeModule) Start(context.Context) error {
	*m.events = append(*m.events, "start:"+m.name)
	return m.startErr
}

func (m *fakeModule) Stop(context.Context) error {
	*m.events = append(*m.events, "stop:"+m.name)
	return m.stopErr
}

func discardLogger() *slog.Logger {
	return observability.NewLogger(io.Discard, slog.LevelDebug, "json")
}

func newApp(t *testing.T, modules ...app.Module) *app.App {
	t.Helper()

	application, err := app.New(discardLogger(), modules...)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	return application
}

func TestStartAndStopRunModulesInOrder(t *testing.T) {
	events := []string{}
	first := &fakeModule{name: "first", events: &events}
	second := &fakeModule{name: "second", events: &events}

	application := newApp(t, first, second)

	if err := application.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	if err := application.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	want := []string{"start:first", "start:second", "stop:second", "stop:first"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestStartRollsBackOnFailure(t *testing.T) {
	events := []string{}
	first := &fakeModule{name: "first", events: &events}
	failing := &fakeModule{name: "failing", events: &events, startErr: errors.New("boom")}
	third := &fakeModule{name: "third", events: &events}

	application := newApp(t, first, failing, third)

	err := application.Start(context.Background())
	if err == nil {
		t.Fatal("Start() must fail when a module fails")
	}

	if !strings.Contains(err.Error(), "failing") {
		t.Fatalf("error = %q, want it to name the failing module", err)
	}

	want := []string{"start:first", "start:failing", "stop:first"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestStopJoinsModuleErrors(t *testing.T) {
	events := []string{}
	first := &fakeModule{name: "first", events: &events, stopErr: errors.New("first failed")}
	second := &fakeModule{name: "second", events: &events, stopErr: errors.New("second failed")}

	application := newApp(t, first, second)

	err := application.Stop(context.Background())
	if err == nil {
		t.Fatal("Stop() must report module failures")
	}

	for _, want := range []string{"first", "second"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to contain %q", err, want)
		}
	}
}

func TestNewRejectsInvalidModules(t *testing.T) {
	events := []string{}

	if _, err := app.New(discardLogger(), nil); err == nil {
		t.Fatal("New() must reject a nil module")
	}

	unnamed := &fakeModule{name: "", events: &events}
	if _, err := app.New(discardLogger(), unnamed); err == nil {
		t.Fatal("New() must reject a module without a name")
	}

	duplicate := &fakeModule{name: "same", events: &events}
	if _, err := app.New(discardLogger(), duplicate, duplicate); err == nil {
		t.Fatal("New() must reject duplicate module names")
	}
}

func TestApplicationServesHealthAndStops(t *testing.T) {
	cfg := config.Default().Server
	cfg.Port = 0

	logger := discardLogger()
	readiness := health.NewRegistry()
	router := server.NewRouter(logger, health.Live(), readiness.Ready())
	httpServer := server.New(cfg, router, logger)

	application := newApp(t, httpServer)

	if err := application.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	response, err := http.Get("http://" + httpServer.Addr() + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz returned error: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if got, want := response.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	if !bytes.Contains(body, []byte(`"status":"up"`)) {
		t.Fatalf("body = %q, want status up", body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := application.Stop(ctx); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
}
