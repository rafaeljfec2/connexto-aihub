package server

import (
	"log/slog"
	"net/http"
)

// NewRouter builds the HTTP handler tree. Health handlers are injected so the
// router stays independent of the health package.
func NewRouter(logger *slog.Logger, live, ready http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", live)
	mux.Handle("GET /readyz", ready)

	return withCorrelationID(logger, withRequestLog(logger, withRecovery(logger, mux)))
}
