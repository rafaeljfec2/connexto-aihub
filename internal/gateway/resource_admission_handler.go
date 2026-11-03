package gateway

import (
	"errors"
	"net/http"
	"time"

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

	decision, err := h.resources.Admit(r.Context(), resources.AdmissionRequest{
		ID:         admissionID,
		ProviderID: providerID,
		ModelID:    modelID,
	})
	if errors.Is(err, resources.ErrLimitExceeded) {
		writeJSONErrorWithReset(w, http.StatusTooManyRequests, resourceLimitExceededType, resourceLimitExceededMsg,
			decision.ResetAt.UTC().Format(time.RFC3339))
		return false
	}
	if errors.Is(err, resources.ErrInvalidInput) {
		writeJSONError(w, http.StatusBadRequest, invalidRequestType, "the routed execution target is invalid")
		return false
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "resource admission failed", "error", err)
		writeJSONError(w, http.StatusServiceUnavailable, resourceAdmissionUnavailableType, resourceAdmissionUnavailableMsg)
		return false
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