package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aihub/internal/execution"
	"aihub/internal/metering"
	"aihub/internal/metering/capture"
	"aihub/internal/observability"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

// StreamInactivityTimeout bounds silence between streamed chunks.
const StreamInactivityTimeout = 2 * time.Minute

const (
	// Keep captured identifiers within the execution-record model_id bound.
	maxObservedModelIDBytes = 4096
)

// NewResponsesHandler creates a standalone execution manager for this handler.
func NewResponsesHandler(client *providers.Client, logger *slog.Logger, inactivity time.Duration, routingDeps *ResponsesRouting) http.Handler {
	manager := newExecutionManager(client, logger, inactivity)
	return NewResponsesHandlerWithManager(client, manager, logger, inactivity, routingDeps, nil, nil, nil)
}

// NewResponsesHandlerWithManager binds the HTTP boundary to the shared manager.
func NewResponsesHandlerWithManager(client *providers.Client, manager *execution.Manager, logger *slog.Logger, inactivity time.Duration, routingDeps *ResponsesRouting, resourceAdmission ResourceAdmitter, concurrencyGate ConcurrencyGate, usage *metering.Service) http.Handler {
	if inactivity <= 0 {
		inactivity = StreamInactivityTimeout
	}
	return &responsesHandler{client: client, manager: manager, logger: logger, inactivity: inactivity, routing: routingDeps, resources: resourceAdmission, concurrency: concurrencyGate, usage: usage}
}

// NewResponsesHandlerWithRegistry resolves each admitted request by provider ID.
func NewResponsesHandlerWithRegistry(registry *providers.Registry, manager *execution.Manager, logger *slog.Logger, inactivity time.Duration, routingDeps *ResponsesRouting, resourceAdmission ResourceAdmitter, concurrencyGate ConcurrencyGate, usage *metering.Service) http.Handler {
	if inactivity <= 0 {
		inactivity = StreamInactivityTimeout
	}
	return &responsesHandler{registry: registry, manager: manager, logger: logger, inactivity: inactivity, routing: routingDeps, resources: resourceAdmission, concurrency: concurrencyGate, usage: usage}
}

type responsesHandler struct {
	client      *providers.Client
	registry    *providers.Registry
	manager     *execution.Manager
	logger      *slog.Logger
	inactivity  time.Duration
	routing     *ResponsesRouting
	resources   ResourceAdmitter
	concurrency ConcurrencyGate
	usage       *metering.Service
}

func (h *responsesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	explicitProvider, err := explicitProviderFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "the provider selector is invalid")
		return
	}

	sessionID, err := providerSessionID(r)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "session id generation failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, executionFailedType, executionFailedMsg)
		return
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "the request body could not be read")
		return
	}
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	requestID := observability.CorrelationIDFrom(r.Context())
	providerID := providers.OpenCodeGoID
	modelID := requestedModelID(payload)
	executionPayload := payload

	var routingDecision routing.RoutingDecision
	pendingRoutingLog := false
	if h.routing != nil {
		var target routing.Target
		var prepared []byte
		var routeErr error
		target, prepared, routingDecision, routeErr = routeResponses(h.routing, h.registry, requestID, explicitProvider, payload)
		logRoutingDecision := func(reachedExecution bool) {
			routing.LogDecision(r.Context(), h.logger, routing.DecisionLogInput{
				Decision:            routingDecision,
				ExplicitProvider:    explicitProvider,
				StructuralRejection: structuralRejectionFromRouteError(routeErr),
				ReachedExecution:    reachedExecution,
			})
		}
		if routeErr != nil {
			logRoutingDecision(false)
			if writeRoutingFailure(w, routeErr) {
				return
			}
			h.logger.ErrorContext(r.Context(), "responses routing failed", "error", routeErr)
			writeJSONError(w, http.StatusInternalServerError, executionFailedType, executionFailedMsg)
			return
		}
		providerID = target.ProviderID
		modelID = target.ModelID
		executionPayload = prepared
		if !h.requireProvider(providerID, w) {
			logRoutingDecision(false)
			return
		}
		pendingRoutingLog = true
	} else {
		providerID, err = requestedProviderID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_request", "the provider selector is invalid")
			return
		}
		if !h.requireProvider(providerID, w) {
			return
		}
	}

	notifyIntegrationRoutingSelected(providerID, modelID)

	executionID, err := execution.NewExecutionID()
	if err != nil {
		h.logger.ErrorContext(r.Context(), "execution id generation failed", "error", err)
		writeJSONError(w, http.StatusServiceUnavailable, executionFailedType, executionFailedMsg)
		return
	}
	if !h.acquireConcurrency(w, r, executionID, providerID, modelID) {
		if pendingRoutingLog {
			h.logRoutingAfterResourceReject(r, true, routingDecision, explicitProvider)
		}
		return
	}
	if !h.admitResources(w, r, providerID, modelID) {
		h.releaseConcurrency(r.Context(), executionID)
		if pendingRoutingLog {
			h.logRoutingAfterResourceReject(r, true, routingDecision, explicitProvider)
		}
		return
	}

	execCtx, cancelExec := context.WithCancel(r.Context())
	defer cancelExec()

	stopRenew := h.startConcurrencyRenewal(execCtx, executionID, cancelExec)
	defer stopRenew()

	streamTap := capture.NewStreamTap()
	responseWriter := &responseSink{writer: w, inactivity: h.inactivity, tap: streamTap}
	result, err := h.manager.Execute(execCtx, execution.Command{
		RequestID:   requestID,
		ExecutionID: executionID,
		Target: execution.Target{
			ProviderID: providerID,
			ModelID:    modelID,
		},
		ProviderSessionID: sessionID,
		Payload:           executionPayload,
		ContentType:       contentType,
	}, responseWriter)
	if h.usage != nil && err == nil {
		h.usage.RecordFromCaptureBestEffort(r.Context(), metering.RecordInput{
			ExecutionID: result.ExecutionID,
			ProviderID:  providerID,
			ModelID:     modelID,
			Capture:     streamTap.Finalize(responseWriter.contentType),
		})
	}
	stopRenew()
	if pendingRoutingLog {
		routing.LogDecision(r.Context(), h.logger, routing.DecisionLogInput{
			Decision:         routingDecision,
			ExplicitProvider: explicitProvider,
			ReachedExecution: err == nil,
		})
	}
	if errors.Is(err, execution.ErrUnavailable) {
		h.releaseConcurrency(r.Context(), executionID)
		writeJSONError(w, http.StatusServiceUnavailable, executionUnavailableType, executionUnavailableMsg)
		return
	}
	if err != nil {
		h.releaseConcurrency(r.Context(), executionID)
		h.logger.ErrorContext(r.Context(), "execution command rejected", "error", err)
		writeJSONError(w, http.StatusInternalServerError, executionFailedType, executionFailedMsg)
		return
	}
	if result.Committed {
		return
	}

	if result.Outcome == execution.OutcomeTimedOut && result.Phase == execution.PhaseOpening {
		writeJSONError(w, http.StatusGatewayTimeout, executionTimeoutType, executionTimeoutMsg)
		return
	}
	if result.Outcome == execution.OutcomeFailed && result.Phase == execution.PhaseOpening {
		writeJSONError(w, http.StatusBadGateway, upstreamUnavailableType, upstreamUnavailableMsg)
	}
}

func structuralRejectionFromRouteError(err error) string {
	if err == nil {
		return ""
	}

	var failure routingHTTPError
	if !errors.As(err, &failure) {
		return ""
	}

	if failure.typ != string(routing.StructuralInvalidRequest) && failure.typ != invalidRequestType {
		return ""
	}

	return failure.typ
}

func requestedModelID(payload []byte) string {
	var request struct {
		Model *string `json:"model"`
	}
	if err := json.Unmarshal(payload, &request); err != nil || request.Model == nil {
		return ""
	}

	modelID := *request.Model
	if strings.TrimSpace(modelID) == "" || len(modelID) > maxObservedModelIDBytes {
		return ""
	}

	return modelID
}
