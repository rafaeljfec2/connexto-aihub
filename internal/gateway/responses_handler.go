package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"aihub/internal/providers"
)

// StreamInactivityTimeout bounds the silence between two streamed chunks.
// It re-arms per chunk, so whole streams may run far longer, and it is an
// inactivity bound rather than a total-duration cap (ADR-008).
const StreamInactivityTimeout = 2 * time.Minute

const (
	streamCopyBufferSize  = 32 << 10
	streamChannelCapacity = 16
)

// sessionHeader carries the provider execution context. Request correlation
// (X-Request-Id) is unrelated and never derives this value.
const sessionHeader = "X-Opencode-Session"

// SessionPrefix keeps generated provider session identifiers recognizable.
const SessionPrefix = "aihub-ses-"

// NewResponsesHandler returns the Responses passthrough handler. The request
// body reaches the upstream byte-for-byte; the adapter never injects
// previous_response_id or performs any protocol translation (D4).
func NewResponsesHandler(client *providers.Client, logger *slog.Logger, inactivity time.Duration) http.Handler {
	if inactivity <= 0 {
		inactivity = StreamInactivityTimeout
	}

	return &responsesHandler{client: client, logger: logger, inactivity: inactivity}
}

type responsesHandler struct {
	client     *providers.Client
	logger     *slog.Logger
	inactivity time.Duration
}

func (h *responsesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.client.Enabled() {
		writeJSONError(w, http.StatusServiceUnavailable, providerUnavailableType, providerUnavailableMsg)

		return
	}

	sessionID, err := resolveSessionID(r)
	if err != nil {
		h.logger.Error("session id generation failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	response, err := h.client.OpenResponses(r.Context(), r.Body, contentType, sessionID)
	if err != nil {
		// Only the wrapped, credential-free message is recorded; the raw
		// transport error is never echoed to the client.
		h.logger.Error("upstream responses request failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, upstreamUnavailableType, upstreamUnavailableMsg)

		return
	}

	defer func() { _ = response.Body.Close() }()

	if upstreamType := response.Header.Get("Content-Type"); upstreamType != "" {
		w.Header().Set("Content-Type", upstreamType)
	}

	w.WriteHeader(response.StatusCode)
	h.copyStream(w, r, response.Body)
}

// copyStream forwards the upstream body chunk by chunk, flushing after every
// chunk and re-arming both deadline mechanisms (ADR-008). Reads happen on a
// dedicated goroutine, so the handler stays responsive to client disconnects
// and inactivity.
func (h *responsesHandler) copyStream(w http.ResponseWriter, r *http.Request, body io.ReadCloser) {
	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Now().Add(h.inactivity))

	quit := make(chan struct{})
	defer close(quit)

	timer := time.NewTimer(h.inactivity)
	defer timer.Stop()

	chunks := startReader(body, quit)

	for {
		select {
		case <-timer.C:
			h.logger.Warn("stream inactivity timeout; aborting passthrough", "path", r.URL.Path)

			return

		case <-r.Context().Done():
			h.logger.Info("client disconnected; aborting passthrough", "path", r.URL.Path)

			return

		case chunk := <-chunks:
			if chunk.err != nil {
				if !errors.Is(chunk.err, io.EOF) {
					h.logger.Warn("upstream stream ended with error", "path", r.URL.Path)
				}

				return
			}

			if _, writeErr := w.Write(chunk.data); writeErr != nil {
				h.logger.Warn("client write failed; aborting passthrough", "path", r.URL.Path)

				return
			}

			if flusher, canFlush := w.(http.Flusher); canFlush {
				flusher.Flush()
			}
			_ = controller.SetWriteDeadline(time.Now().Add(h.inactivity))
			rearmTimer(timer, h.inactivity)
		}
	}
}

// chunkResult is one body read consumed by the copy loop.
type chunkResult struct {
	data []byte
	err  error
}

// startReader pumps the upstream body into a bounded channel. Sending
// respects quit, so the reader goroutine always exits and never leaks.
func startReader(body io.Reader, quit <-chan struct{}) <-chan chunkResult {
	chunks := make(chan chunkResult, streamChannelCapacity)

	go func() {
		buffer := make([]byte, streamCopyBufferSize)
		for {
			n, err := body.Read(buffer)
			if n > 0 {
				result := make([]byte, n)
				copy(result, buffer[:n])

				select {
				case chunks <- chunkResult{data: result}:
				case <-quit:
					return
				}
			}

			if err != nil {
				select {
				case chunks <- chunkResult{err: err}:
				case <-quit:
				}

				return
			}
		}
	}()

	return chunks
}

// rearmTimer restarts the read watchdog. Only the copy loop touches the timer,
// so the stop/drain/reset sequence is race-free.
func rearmTimer(timer *time.Timer, inactivity time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}

	timer.Reset(inactivity)
}

// resolveSessionID keeps request correlation independent of the provider
// execution context (D3): a supplied session header wins, otherwise a fresh
// session identifier is generated with its own entropy source.
func resolveSessionID(r *http.Request) (string, error) {
	if supplied := r.Header.Get(sessionHeader); supplied != "" {
		return supplied, nil
	}

	return newSessionID()
}

func newSessionID() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate provider session id: %w", err)
	}

	return SessionPrefix + hex.EncodeToString(random[:]), nil
}
