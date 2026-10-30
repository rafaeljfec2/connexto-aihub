package server_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/health"
	"aihub/internal/observability"
	"aihub/internal/server"
)

func testLogger(buffer *bytes.Buffer) *slog.Logger {
	return observability.NewLogger(buffer, slog.LevelDebug, "json")
}

func newRouterTestHandler(t *testing.T) (*http.Handler, *bytes.Buffer) {
	t.Helper()

	buffer := &bytes.Buffer{}
	registry := health.NewRegistry()
	handler := server.NewRouter(testLogger(buffer), health.Live(), registry.Ready())

	return &handler, buffer
}

func TestRouterServesHealthEndpoints(t *testing.T) {
	handler, _ := newRouterTestHandler(t)

	tests := []struct {
		name string
		path string
	}{
		{name: "liveness", path: "/healthz"},
		{name: "readiness", path: "/readyz"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			(*handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))

			if got, want := recorder.Code, http.StatusOK; got != want {
				t.Fatalf("status = %d, want %d", got, want)
			}
		})
	}
}

func TestRouterRejectsUnsupportedMethod(t *testing.T) {
	handler, _ := newRouterTestHandler(t)

	recorder := httptest.NewRecorder()
	(*handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/healthz", nil))

	if got, want := recorder.Code, http.StatusMethodNotAllowed; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}

func TestRouterReturnsNotFoundForUnknownPath(t *testing.T) {
	handler, _ := newRouterTestHandler(t)

	recorder := httptest.NewRecorder()
	(*handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	if got, want := recorder.Code, http.StatusNotFound; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}

func TestRouterGeneratesCorrelationID(t *testing.T) {
	handler, _ := newRouterTestHandler(t)

	recorder := httptest.NewRecorder()
	(*handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	id := recorder.Header().Get("X-Request-Id")
	if !observability.IsValidCorrelationID(id) {
		t.Fatalf("X-Request-Id = %q, want a valid generated identifier", id)
	}
}

func TestRouterPropagatesValidCorrelationID(t *testing.T) {
	handler, _ := newRouterTestHandler(t)
	correlationID := "client-request-0001"

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-Id", correlationID)

	recorder := httptest.NewRecorder()
	(*handler).ServeHTTP(recorder, request)

	if got := recorder.Header().Get("X-Request-Id"); got != correlationID {
		t.Fatalf("X-Request-Id = %q, want %q", got, correlationID)
	}
}

func TestRouterReplacesInvalidCorrelationID(t *testing.T) {
	handler, _ := newRouterTestHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-Id", "invalid id with spaces")

	recorder := httptest.NewRecorder()
	(*handler).ServeHTTP(recorder, request)

	id := recorder.Header().Get("X-Request-Id")
	if id == "invalid id with spaces" {
		t.Fatal("invalid correlation id must be replaced")
	}

	if !observability.IsValidCorrelationID(id) {
		t.Fatalf("X-Request-Id = %q, want a valid generated identifier", id)
	}
}

func TestRouterLogsCorrelationIDAndOmitsCredentials(t *testing.T) {
	handler, buffer := newRouterTestHandler(t)
	const credential = "Bearer super-secret-token"

	request := httptest.NewRequest(http.MethodGet, "/healthz?api_key=super-secret-token", nil)
	request.Header.Set("Authorization", credential)
	request.Header.Set("X-Request-Id", "log-check-000001")

	recorder := httptest.NewRecorder()
	(*handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	logs := buffer.String()

	if !strings.Contains(logs, "log-check-000001") {
		t.Fatalf("logs must contain the correlation id, got %q", logs)
	}

	if strings.Contains(logs, "super-secret-token") {
		t.Fatalf("logs must not contain credentials, got %q", logs)
	}

	if strings.Contains(logs, "Bearer") {
		t.Fatalf("logs must not contain authorization headers, got %q", logs)
	}
}

func TestRouterRecoversFromPanics(t *testing.T) {
	buffer := &bytes.Buffer{}
	panicHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	handler := server.NewRouter(testLogger(buffer), panicHandler, panicHandler)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if got, want := recorder.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	if !strings.Contains(buffer.String(), "panic recovered") {
		t.Fatalf("logs must record the recovered panic, got %q", buffer.String())
	}
}

func TestServerStartAndStop(t *testing.T) {
	cfg := config.Default().Server
	cfg.Port = 0
	buffer := &bytes.Buffer{}

	instance := server.New(cfg, http.NotFoundHandler(), testLogger(buffer))
	if err := instance.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	response, err := http.Get("http://" + instance.Addr() + "/healthz")
	if err != nil {
		t.Fatalf("GET returned error: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if got, want := response.StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := instance.Stop(ctx); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
}

func TestServerStartFailsWhenAddressIsInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	address := listener.Addr().(*net.TCPAddr)

	cfg := config.Default().Server
	cfg.Host = "127.0.0.1"
	cfg.Port = address.Port

	instance := server.New(cfg, http.NotFoundHandler(), testLogger(&bytes.Buffer{}))

	if err := instance.Start(context.Background()); err == nil {
		t.Fatal("Start() must fail when the address is already in use")
	}
}

func TestServerStopCompletesInFlightRequests(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		w.WriteHeader(http.StatusOK)
	})

	cfg := config.Default().Server
	cfg.Port = 0

	instance := server.New(cfg, handler, testLogger(&bytes.Buffer{}))
	if err := instance.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	responseErr := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + instance.Addr() + "/slow")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		responseErr <- err
	}()

	<-requestStarted

	stopErr := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stopErr <- instance.Stop(ctx)
	}()

	close(releaseRequest)

	if err := <-responseErr; err != nil {
		t.Fatalf("in-flight request failed during shutdown: %v", err)
	}

	if err := <-stopErr; err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
}

func TestRouterKeepsPartialResponseWhenHandlerPanics(t *testing.T) {
	buffer := &bytes.Buffer{}
	partialHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial body"))
		panic("boom after write")
	})

	handler := server.NewRouter(testLogger(buffer), partialHandler, partialHandler)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d committed status", got, want)
	}

	if got, want := recorder.Body.String(), "partial body"; got != want {
		t.Fatalf("body = %q, want %q without appended error text", got, want)
	}

	if !strings.Contains(buffer.String(), "panic recovered") {
		t.Fatalf("logs must record the recovered panic, got %q", buffer.String())
	}
}

func TestServerStartFailsWithCancelledContext(t *testing.T) {
	cfg := config.Default().Server
	cfg.Port = 0

	instance := server.New(cfg, http.NotFoundHandler(), testLogger(&bytes.Buffer{}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := instance.Start(ctx); err == nil {
		t.Fatal("Start() must fail when the context is already cancelled")
	}
}

func TestServerWarnsWhenBoundToNonLoopbackAddress(t *testing.T) {
	cfg := config.Default().Server
	cfg.Host = "0.0.0.0"
	cfg.Port = 0

	buffer := &bytes.Buffer{}
	instance := server.New(cfg, http.NotFoundHandler(), testLogger(buffer))

	if err := instance.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := instance.Stop(ctx); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	if !strings.Contains(buffer.String(), "non-loopback") {
		t.Fatalf("logs must warn about the non-loopback bind, got %q", buffer.String())
	}
}

func TestServerStopReturnsErrorWhenDeadlineExpires(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		w.WriteHeader(http.StatusOK)
	})

	cfg := config.Default().Server
	cfg.Port = 0

	instance := server.New(cfg, handler, testLogger(&bytes.Buffer{}))
	if err := instance.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)

		response, err := http.Get("http://" + instance.Addr() + "/hang")
		if err != nil {
			return
		}

		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()

	<-requestStarted

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := instance.Stop(ctx); err == nil {
		t.Fatal("Stop() must return an error when the deadline expires with in-flight requests")
	}

	close(releaseRequest)
	<-requestDone
}
