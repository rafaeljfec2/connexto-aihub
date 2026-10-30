// Package gateway exposes AI Hub passthrough handlers to the provider:
// Responses execution, model catalog retrieval and provider diagnostics.
package gateway

import (
	"encoding/json"
	"net/http"
)

// writeJSONError answers with the OpenAI error envelope the gateway uses for
// its own failures. Bodies are generic; details live in logs only.
type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Failure types surfaced by the gateway itself.
const (
	providerUnavailableType = "provider_unavailable"
	upstreamUnavailableType = "upstream_unavailable"
	providerUnavailableMsg  = "provider opencode-go is not configured"
	upstreamUnavailableMsg  = "the upstream provider request failed"
)

func writeJSONError(w http.ResponseWriter, status int, failureType, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(errorEnvelope{Error: errorPayload{
		Type:    failureType,
		Message: message,
	}}); err != nil {
		return
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return
	}
}
