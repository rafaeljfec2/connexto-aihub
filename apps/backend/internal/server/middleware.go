package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"aihub/internal/observability"
)

// correlationHeader carries the request identifier in both directions.
const correlationHeader = "X-Request-Id"

type accessLogManagedContextKey struct{}

// withCorrelationID reuses a valid inbound identifier or generates a new one,
// stores it in the request context and echoes it in the response.
func withCorrelationID(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(correlationHeader)
		if !observability.IsValidCorrelationID(id) {
			generated, err := observability.NewCorrelationID()
			if err != nil {
				logger.Error("correlation id generation failed", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			id = generated
		}

		w.Header().Set(correlationHeader, id)
		next.ServeHTTP(w, r.WithContext(observability.WithCorrelationID(r.Context(), id)))
	})
}

// withRequestLog records one structured entry per request. It intentionally
// omits headers and query strings so credentials never reach the logs.
func withRequestLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(accessLogManagedContextKey{}) == true {
			next.ServeHTTP(w, r)
			return
		}

		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		logger.LogAttrs(r.Context(), level, "http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Int("bytes", recorder.bytes),
			slog.Duration("duration", time.Since(started)),
			slog.String("correlation_id", observability.CorrelationIDFrom(r.Context())),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}

// withRecovery converts handler panics into a 500 response and a log entry.
func withRecovery(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			logger.Error("panic recovered",
				"panic", fmt.Sprint(recovered),
				"path", r.URL.Path,
				"correlation_id", observability.CorrelationIDFrom(r.Context()),
				"stack", string(debug.Stack()),
			)

			// Once the status is committed the response can no longer be
			// replaced without corrupting the body already sent.
			if recorder, ok := w.(*responseRecorder); ok && recorder.status != 0 {
				return
			}

			http.Error(w, "internal error", http.StatusInternalServerError)
		}()

		next.ServeHTTP(w, r)
	})
}

// responseRecorder captures the status code and body size for logging while
// preserving the optional interfaces used by streaming responses.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}

	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	written, err := r.ResponseWriter.Write(body)
	r.bytes += written

	return written, err
}

// Flush preserves streaming behaviour for handlers that flush explicitly.
func (r *responseRecorder) Flush() {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap exposes the original writer to http.ResponseController.
func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
