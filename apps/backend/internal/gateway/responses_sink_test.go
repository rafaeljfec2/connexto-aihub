package gateway

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestResponseSinkReturnsFlushError(t *testing.T) {
	flushErr := errors.New("client flush failed")
	writer := &flushErrorWriter{header: make(http.Header), flushErr: flushErr}
	sink := &responseSink{writer: writer, inactivity: time.Second}

	if err := sink.WriteChunk([]byte("event: done\n\n")); !errors.Is(err, flushErr) {
		t.Fatalf("WriteChunk() error = %v, want %v", err, flushErr)
	}
	if writer.flushErrorCalls != 1 {
		t.Fatalf("FlushError() calls = %d, want 1", writer.flushErrorCalls)
	}
}

type flushErrorWriter struct {
	header          http.Header
	flushErr        error
	flushErrorCalls int
}

func (w *flushErrorWriter) Header() http.Header       { return w.header }
func (*flushErrorWriter) Write(p []byte) (int, error) { return len(p), nil }
func (*flushErrorWriter) WriteHeader(int)             {}
func (*flushErrorWriter) Flush()                      {}
func (w *flushErrorWriter) FlushError() error {
	w.flushErrorCalls++
	return w.flushErr
}
