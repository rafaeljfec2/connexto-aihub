package server_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aihub/internal/health"
	"aihub/internal/server"
)

func TestPipelinePreservesSSEStreaming(t *testing.T) {
	const (
		firstEvent  = "event: response.created\ndata: {}\n\n"
		secondEvent = "event: response.completed\ndata: {}\n\n"
	)

	firstSent := make(chan struct{})
	releaseSecond := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("streaming writer must support flush")
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, firstEvent)
		flusher.Flush()
		close(firstSent)
		<-releaseSecond
		_, _ = io.WriteString(w, secondEvent)
		flusher.Flush()
	})

	router := server.NewRouter(testLogger(&bytes.Buffer{}), testPolicy(t), health.Live(), health.NewRegistry().Ready(),
		server.Route{Method: http.MethodGet, Pattern: "/v1/stream", Handler: handler},
	)
	instance := httptest.NewServer(router)
	t.Cleanup(instance.Close)

	request, err := http.NewRequest(http.MethodGet, instance.URL+"/v1/stream", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Authorization", authorization())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET /v1/stream: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	type readResult struct {
		data []byte
		err  error
	}
	firstRead := make(chan readResult, 1)
	go func() {
		buffer := make([]byte, len(firstEvent))
		_, readErr := io.ReadFull(response.Body, buffer)
		firstRead <- readResult{data: buffer, err: readErr}
	}()

	select {
	case result := <-firstRead:
		if result.err != nil {
			t.Fatalf("read first event: %v", result.err)
		}
		if string(result.data) != firstEvent {
			t.Fatalf("first event = %q, want %q", result.data, firstEvent)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first SSE event was buffered by the access middleware")
	}

	close(releaseSecond)

	rest, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read remaining stream: %v", err)
	}

	if string(rest) != secondEvent {
		t.Fatalf("remaining stream = %q, want %q", rest, secondEvent)
	}
}

func TestPipelinePreservesResponseController(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, ok := w.(interface{ Unwrap() http.ResponseWriter }); !ok {
			t.Error("writer must expose Unwrap for ResponseController")
		}

		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("ResponseController.Flush() returned error: %v", err)
		}

		_, _ = io.WriteString(w, "ok")
	})

	router := server.NewRouter(testLogger(&bytes.Buffer{}), testPolicy(t), health.Live(), health.NewRegistry().Ready(),
		server.Route{Method: http.MethodGet, Pattern: "/v1/stream", Handler: handler},
	)

	request := httptest.NewRequest(http.MethodGet, "/v1/stream", nil)
	request.Header.Set("Authorization", authorization())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if recorder.Body.String() != "ok" {
		t.Fatalf("body = %q, want ok", recorder.Body.String())
	}
}
