package gateway

import (
	"errors"
	"net/http"
	"strings"

	"aihub/internal/providers"
)

const (
	ProviderHeader               = "X-AIHub-Provider"
	responseProviderNotFoundType = "provider_not_found"
	responseProviderNotFoundMsg  = "provider is not available"
	responseProviderUnavailable  = "the selected provider is unavailable"
)

var errInvalidProviderSelector = errors.New("invalid provider selector")

func requestedProviderID(request *http.Request) (string, error) {
	values := request.Header.Values(ProviderHeader)
	if len(values) == 0 || len(values) == 1 && strings.TrimSpace(values[0]) == "" {
		return providers.OpenCodeGoID, nil
	}
	if len(values) != 1 || len(values[0]) > 128 {
		return "", errInvalidProviderSelector
	}
	return values[0], nil
}

func (h *responsesHandler) requireProvider(providerID string, writer http.ResponseWriter) bool {
	if h.registry == nil {
		if providerID != providers.OpenCodeGoID {
			writeJSONError(writer, http.StatusNotFound, responseProviderNotFoundType, responseProviderNotFoundMsg)
			return false
		}
		if !h.client.Enabled() {
			writeJSONError(writer, http.StatusServiceUnavailable, providerUnavailableType, providerUnavailableMsg)
			return false
		}
		return true
	}
	adapter, ok := h.registry.Resolve(providerID)
	if !ok {
		writeJSONError(writer, http.StatusNotFound, responseProviderNotFoundType, responseProviderNotFoundMsg)
		return false
	}
	if _, ok := adapter.(providers.ResponsesAdapter); !ok {
		writeJSONError(writer, http.StatusNotFound, responseProviderNotFoundType, responseProviderNotFoundMsg)
		return false
	}
	if !adapter.Enabled() {
		writeJSONError(writer, http.StatusServiceUnavailable, providerUnavailableType, responseProviderUnavailable)
		return false
	}
	return true
}
