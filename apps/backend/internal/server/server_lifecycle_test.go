package server_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/server"
)

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
