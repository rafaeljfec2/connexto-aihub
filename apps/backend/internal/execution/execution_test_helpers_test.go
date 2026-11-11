package execution

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type sinkProbe struct {
	mu          sync.Mutex
	status      int
	contentType string
	chunks      []string
	writeErr    error
	wrote       chan string
}

func (s *sinkProbe) Commit(status int, contentType string) {
	s.mu.Lock()
	s.status = status
	s.contentType = contentType
	s.mu.Unlock()
}

func (s *sinkProbe) WriteChunk(chunk []byte) error {
	s.mu.Lock()
	s.chunks = append(s.chunks, string(chunk))
	s.mu.Unlock()
	if s.wrote != nil {
		s.wrote <- string(chunk)
	}
	return s.writeErr
}

func (s *sinkProbe) snapshot() (int, string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, s.contentType, append([]string(nil), s.chunks...)
}

type blockingBody struct {
	first       string
	firstSent   bool
	readStarted chan struct{}
	closed      chan struct{}
	readOnce    sync.Once
	closeOnce   sync.Once
	closeCount  atomic.Int32
}

func newBlockingBody(first string) *blockingBody {
	return &blockingBody{
		first:       first,
		readStarted: make(chan struct{}),
		closed:      make(chan struct{}),
	}
}

func (b *blockingBody) Read(buffer []byte) (int, error) {
	if !b.firstSent && b.first != "" {
		b.firstSent = true
		return copy(buffer, b.first), nil
	}
	b.readOnce.Do(func() { close(b.readStarted) })
	<-b.closed
	return 0, errors.New("body closed")
}

func (b *blockingBody) Close() error {
	b.closeCount.Add(1)
	b.closeOnce.Do(func() { close(b.closed) })
	return nil
}

func newTestManager(inactivity time.Duration, opener OpenResponses) *Manager {
	return NewManager(nil, "opencode-go", opener, inactivity)
}

func validCommand() Command {
	return Command{
		RequestID:         "corr_test_123",
		Target:            Target{ProviderID: "opencode-go"},
		ProviderSessionID: "aihub-ses-test",
		Payload:           []byte("opaque payload"),
		ContentType:       "application/json",
	}
}

func responseFor(body io.ReadCloser, status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       body,
	}
}

func newEOFBody(contents string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(contents))
}

func awaitSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}
