package gateway

import (
	"io"
	"log/slog"
	"net/http"

	"aihub/internal/providers"
)

// NewModelsHandler exposes the provider model catalog at the vendor-specific
// path verified in the spike. Catalog payloads are passed through unchanged.
func NewModelsHandler(client *providers.Client, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !client.Enabled() {
			writeJSONError(w, http.StatusServiceUnavailable, providerUnavailableType, providerUnavailableMsg)

			return
		}

		response, err := client.ListModels(r.Context())
		if err != nil {
			logger.Error("provider models request failed", "error", err)
			writeJSONError(w, http.StatusBadGateway, upstreamUnavailableType, upstreamUnavailableMsg)

			return
		}

		defer func() { _ = response.Body.Close() }()

		if contentType := response.Header.Get("Content-Type"); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		w.WriteHeader(response.StatusCode)

		if _, err := io.Copy(w, response.Body); err != nil {
			logger.Warn("upstream catalog copy failed", "error", err)
		}
	})
}
