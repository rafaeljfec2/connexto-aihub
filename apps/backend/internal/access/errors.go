// Package access implements the AI Hub gateway trust boundary: static client
// authentication, Host/Origin policy, bounded request admission and request
// identity. It depends only on the Go standard library.
package access

import (
	"encoding/json"
	"net/http"
)

// Stable local error types.
const (
	ErrorInvalidRequest       = "invalid_request"
	ErrorInternal             = "internal_error"
	ErrorUnauthorized         = "unauthorized"
	ErrorForbidden            = "forbidden"
	ErrorPayloadTooLarge      = "payload_too_large"
	ErrorUnsupportedMediaType = "unsupported_media_type"
)

const errorContentType = "application/json; charset=utf-8"

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, errorType, message string) {
	w.Header().Set("Content-Type", errorContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ai-hub"`)
	}

	payload, err := json.Marshal(errorEnvelope{Error: errorDetail{Type: errorType, Message: message}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func forbidden(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, ErrorForbidden, "request origin or authority is not allowed")
}

func unauthorized(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, ErrorUnauthorized, "a valid gateway bearer token is required")
}

// WriteUnauthorized emits a generic unauthorized JSON envelope.
func WriteUnauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		unauthorized(w)
		return
	}
	writeError(w, http.StatusUnauthorized, ErrorUnauthorized, message)
}

// WriteInvalidRequest emits a bad request JSON envelope.
func WriteInvalidRequest(w http.ResponseWriter, message string) {
	invalidRequest(w, message)
}

// WriteInternalError emits an internal error JSON envelope.
func WriteInternalError(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, ErrorInternal, "an internal error occurred")
}

// WriteMethodNotAllowed emits HTTP 405 without a JSON body.
func WriteMethodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, POST")
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func invalidRequest(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, ErrorInvalidRequest, message)
}

func payloadTooLarge(w http.ResponseWriter) {
	writeError(w, http.StatusRequestEntityTooLarge, ErrorPayloadTooLarge, "request body exceeds the configured limit")
}

func unsupportedMediaType(w http.ResponseWriter) {
	writeError(w, http.StatusUnsupportedMediaType, ErrorUnsupportedMediaType, "request media type or encoding is not supported")
}
