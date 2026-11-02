package gateway

import (
	"io"
	"log/slog"
	"net/http"

	"aihub/internal/providers"
)

// NewRegistryModelsHandler selects the native model-list endpoint by provider ID.
func NewRegistryModelsHandler(registry *providers.Registry, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		providerID, err := requestedProviderID(request)
		if err != nil {
			writeJSONError(writer, http.StatusBadRequest, "invalid_request", "the provider selector is invalid")
			return
		}
		if registry == nil {
			writeJSONError(writer, http.StatusNotFound, responseProviderNotFoundType, responseProviderNotFoundMsg)
			return
		}
		adapter, ok := registry.Resolve(providerID)
		if !ok {
			writeJSONError(writer, http.StatusNotFound, responseProviderNotFoundType, responseProviderNotFoundMsg)
			return
		}
		models, ok := adapter.(providers.ModelListAdapter)
		if !ok {
			writeJSONError(writer, http.StatusNotFound, responseProviderNotFoundType, responseProviderNotFoundMsg)
			return
		}
		if !adapter.Enabled() {
			writeJSONError(writer, http.StatusServiceUnavailable, providerUnavailableType, responseProviderUnavailable)
			return
		}
		response, err := models.ListModels(request.Context())
		if err != nil || response == nil {
			logger.ErrorContext(request.Context(), "provider model list failed", "provider", providerID, "error", err)
			writeJSONError(writer, http.StatusBadGateway, upstreamUnavailableType, upstreamUnavailableMsg)
			return
		}
		defer func() { _ = response.Body.Close() }()
		if contentType := response.Header.Get("Content-Type"); contentType != "" {
			writer.Header().Set("Content-Type", contentType)
		}
		writer.WriteHeader(response.StatusCode)
		if _, err := io.Copy(writer, response.Body); err != nil {
			logger.WarnContext(request.Context(), "provider model list response copy failed", "provider", providerID)
		}
	})
}
