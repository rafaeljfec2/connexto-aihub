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
	ResetAt string `json:"reset_at,omitempty"`
}

// Failure types surfaced by the gateway itself.
const (
	providerUnavailableType  = "provider_unavailable"
	upstreamUnavailableType  = "upstream_unavailable"
	executionUnavailableType = "execution_unavailable"
	executionTimeoutType     = "execution_timeout"
	executionFailedType      = "execution_failed"
	providerUnavailableMsg   = "provider opencode-go is not configured"
	upstreamUnavailableMsg   = "the upstream provider request failed"
	executionUnavailableMsg  = "the execution manager is shutting down"
	executionTimeoutMsg              = "the upstream provider did not respond in time"
	executionFailedMsg               = "the execution could not be started"
	resourceLimitExceededType        = "resource_limit_exceeded"
	resourceAdmissionUnavailableType = "resource_admission_unavailable"
	resourceLimitExceededMsg         = "the configured daily request limit has been reached"
	resourceAdmissionUnavailableMsg  = "resource admission could not be completed safely"
)

func writeJSONError(w http.ResponseWriter, status int, failureType, message string) {
	writeJSONErrorWithReset(w, status, failureType, message, "")
}

func writeJSONErrorWithReset(w http.ResponseWriter, status int, failureType, message, resetAt string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	payload := errorPayload{Type: failureType, Message: message}
	if resetAt != "" {
		payload.ResetAt = resetAt
	}

	if err := json.NewEncoder(w).Encode(errorEnvelope{Error: payload}); err != nil {
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
