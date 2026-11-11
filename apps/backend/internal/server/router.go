package server

import (
	"log/slog"
	"net/http"

	"aihub/internal/access"
)

// Route associates an HTTP method and path pattern with an injected handler,
// keeping the router independent of gateway and provider packages.
type Route struct {
	Method    string
	Pattern   string
	Handler   http.Handler
	Admission access.Admission
}

// NewRouter builds the HTTP handler tree. Health handlers and additional
// application routes are injected so the router stays independent of their
// implementations.
func NewRouter(logger *slog.Logger, policy access.Policy, live, ready http.Handler, additional ...Route) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", live)
	mux.Handle("GET /readyz", ready)

	for _, route := range additional {
		mux.Handle(route.Method+" "+route.Pattern, route.Admission.Apply(route.Handler))
	}

	return withCorrelationID(logger, withRequestLog(logger, withRecovery(logger, policy.Authorize(mux))))
}
