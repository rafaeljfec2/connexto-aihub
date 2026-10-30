package gateway

import (
	"net/http"

	"aihub/internal/providers"
)

// NewProvidersHandler exposes provider configuration and availability state.
// It always answers 200: a disabled or unhealthy provider is a diagnostic
// fact, never an application failure (D-R).
func NewProvidersHandler(statuses func() []providers.Status) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"providers": statuses()})
	})
}
