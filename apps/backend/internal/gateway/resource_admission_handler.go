package gateway

import (
	"errors"
	"net/http"

	"aihub/internal/resources"
	"aihub/internal/routing"
)

func (h *responsesHandler) admitResources(w http.ResponseWriter, r *http.Request, providerID, modelID string) bool {
	if h.resources == nil {
		return true
	}

	admissionID, err := resources.NewAdmissionID()
	if err != nil {
		h.logger.ErrorContext(r.Context(), "resource admission id generation failed", "error", err)
		writeJSONError(w, http.StatusServiceUnavailable, resourceAdmissionUnavailableType, resourceAdmissionUnavailableMsg)
		return false
	}

	if IntegrationTestHooks.OnResourceAdmissionStarted != nil {
		IntegrationTestHooks.OnResourceAdmissionStarted(providerID, modelID)
	}

	decision, err := h.resources.Admit(r.Context(), resources.AdmissionRequest{
		ID:         admissionID,
		ProviderID: providerID,
		ModelID:    modelID,
	})
	if errors.Is(err, resources.ErrLimitExceeded) {
		// Completions and responses are not rejected with 429 when a daily
		// policy is full. Codex treats that status as a retryable rate limit.
		if IntegrationTestHooks.OnResourceAdmissionRejected != nil {
			IntegrationTestHooks.OnResourceAdmissionRejected(providerID, modelID)
		}
		h.logger.WarnContext(r.Context(), "resource limit reached; execution endpoint proceeds",
			"provider_id", providerID, "model_id", modelID)
		if IntegrationTestHooks.OnResourceAdmissionAccepted != nil {
			IntegrationTestHooks.OnResourceAdmissionAccepted(providerID, modelID)
		}
		return true
	}
	if errors.Is(err, resources.ErrInvalidInput) {
		if IntegrationTestHooks.OnResourceAdmissionFailed != nil {
			IntegrationTestHooks.OnResourceAdmissionFailed(providerID, modelID)
		}
		writeJSONError(w, http.StatusBadRequest, invalidRequestType, "the routed execution target is invalid")
		return false
	}
	if err != nil {
		if IntegrationTestHooks.OnResourceAdmissionFailed != nil {
			IntegrationTestHooks.OnResourceAdmissionFailed(providerID, modelID)
		}
		h.logger.ErrorContext(r.Context(), "resource admission failed", "error", err)
		writeJSONError(w, http.StatusServiceUnavailable, resourceAdmissionUnavailableType, resourceAdmissionUnavailableMsg)
		return false
	}
	if IntegrationTestHooks.OnResourceAdmissionAccepted != nil {
		IntegrationTestHooks.OnResourceAdmissionAccepted(providerID, modelID)
	}
	_ = decision
	return true
}

func (h *responsesHandler) logRoutingAfterResourceReject(r *http.Request, pending bool, decision routing.RoutingDecision, explicitProvider string) {
	if !pending {
		return
	}
	routing.LogDecision(r.Context(), h.logger, routing.DecisionLogInput{
		Decision:         decision,
		ExplicitProvider: explicitProvider,
		ReachedExecution: false,
	})
}