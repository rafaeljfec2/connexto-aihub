package server_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/server"
)

func TestFiberServerStreamsEventsProgressively(t *testing.T) {
	const firstEvent = "event: response.created\ndata: {}\n\n"

	firstSent := make(chan struct{})
	releaseSecond := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseSecond) }) }
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, firstEvent)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		close(firstSent)
		<-releaseSecond
		_, _ = io.WriteString(w, "event: response.completed\ndata: {}\n\n")
	})

	cfg := config.Default().Server
	cfg.Port = 0
	instance := server.New(cfg, handler, testLogger(&bytes.Buffer{}))
	if err := instance.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	t.Cleanup(func() {
		release()
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = instance.Stop(stopCtx)
	})

	request, err := http.NewRequest(http.MethodGet, "http://"+instance.Addr()+"/stream", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", response.Header.Get("Content-Type"))
	}

	firstRead := make(chan []byte, 1)
	go func() {
		body := make([]byte, len(firstEvent))
		_, _ = io.ReadFull(response.Body, body)
		firstRead <- body
	}()
	select {
	case body := <-firstRead:
		if string(body) != firstEvent {
			t.Fatalf("first event = %q, want %q", body, firstEvent)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first SSE event was not flushed progressively")
	}

	select {
	case <-firstSent:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not reach the first event")
	}
	release()
	remaining, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read remaining stream: %v", err)
	}
	if string(remaining) != "event: response.completed\ndata: {}\n\n" {
		t.Fatalf("remaining stream = %q", remaining)
	}
}

func TestFiberServerAcceptsRequestBodiesAboveFourMiB(t *testing.T) {
	const size = (4 << 20) + 1

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(body) != size {
			http.Error(w, "unexpected body length", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	instance := startFiberServer(t, handler)
	response := postBody(t, instance.Addr(), bytes.Repeat([]byte("a"), size))
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 204, body = %s", response.StatusCode, payload)
	}
}

func TestFiberServerRejectsBodiesOverTheConfiguredLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	cfg := config.Default().Server
	cfg.Port = 0
	instance := server.New(cfg, handler, testLogger(&bytes.Buffer{}), server.WithBodyLimit(1024))
	startInstance(t, instance)

	response := postBody(t, instance.Addr(), bytes.Repeat([]byte("a"), 2048))
	defer response.Body.Close()

	if response.StatusCode != http.StatusRequestEntityTooLarge {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 413, body = %s", response.StatusCode, payload)
	}
}

func startFiberServer(t *testing.T, handler http.Handler) *server.Server {
	t.Helper()

	cfg := config.Default().Server
	cfg.Port = 0
	instance := server.New(cfg, handler, testLogger(&bytes.Buffer{}))
	startInstance(t, instance)
	return instance
}

func startInstance(t *testing.T, instance *server.Server) {
	t.Helper()

	if err := instance.Start(context.Background()); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = instance.Stop(stopCtx)
	})
}

func postBody(t *testing.T, addr string, body []byte) *http.Response {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, "http://"+addr+"/v1/responses", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.ContentLength = int64(len(body))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /v1/responses: %v", err)
	}
	return response
}
