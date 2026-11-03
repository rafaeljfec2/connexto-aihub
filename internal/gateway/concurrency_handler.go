package gateway

import (
	"context"
	"errors"
	"net/http"

	"aihub/internal/concurrency"
)

const (
	concurrencyLimitExceededType        = "concurrency_limit_exceeded"
	concurrencyAdmissionUnavailableType = "concurrency_admission_unavailable"
	concurrencyLimitExceededMsg         = "the configured concurrency limit has been reached"
	concurrencyAdmissionUnavailableMsg  = "concurrency admission could not be completed safely"
)

func (h *responsesHandler) acquireConcurrency(w http.ResponseWriter, r *http.Request, executionID, providerID, modelID string) bool {
	if h.concurrency == nil {
		return true
	}
	if IntegrationTestHooks.OnConcurrencyAcquireStarted != nil {
		IntegrationTestHooks.OnConcurrencyAcquireStarted(executionID, providerID, modelID)
	}
	_, err := h.concurrency.Acquire(r.Context(), concurrency.AcquireRequest{
		ExecutionID: executionID,
		ProviderID:  providerID,
		ModelID:     modelID,
	})
	if errors.Is(err, concurrency.ErrLimitExceeded) {
		if IntegrationTestHooks.OnConcurrencyAcquireRejected != nil {
			IntegrationTestHooks.OnConcurrencyAcquireRejected(executionID, providerID, modelID)
		}
		writeJSONError(w, http.StatusTooManyRequests, concurrencyLimitExceededType, concurrencyLimitExceededMsg)
		return false
	}
	if err != nil {
		if IntegrationTestHooks.OnConcurrencyAcquireFailed != nil {
			IntegrationTestHooks.OnConcurrencyAcquireFailed(executionID, providerID, modelID)
		}
		h.logger.ErrorContext(r.Context(), "concurrency acquire failed", "error", err, "execution_id", executionID)
		writeJSONError(w, http.StatusServiceUnavailable, concurrencyAdmissionUnavailableType, concurrencyAdmissionUnavailableMsg)
		return false
	}
	if IntegrationTestHooks.OnConcurrencyAcquireAccepted != nil {
		IntegrationTestHooks.OnConcurrencyAcquireAccepted(executionID, providerID, modelID)
	}
	return true
}

func (h *responsesHandler) releaseConcurrency(ctx context.Context, executionID string) {
	if h.concurrency == nil || executionID == "" {
		return
	}
	if err := h.concurrency.Release(ctx, executionID); err != nil {
		h.logger.WarnContext(ctx, "concurrency release failed", "error", err, "execution_id", executionID)
	}
}
