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
