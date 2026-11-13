package gateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"aihub/internal/execution"
	"aihub/internal/generations"
	"aihub/internal/metering"
	"aihub/internal/metering/capture"
	"aihub/internal/observability"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

// StreamInactivityTimeout bounds silence between upstream chunks on one turn.
// SSE pings keep the client attached during that window. A multi-hour agent
// loop is a sequence of turns, so this ceiling has to cover a long generation
// rather than a single model token.
const StreamInactivityTimeout = 12 * time.Hour

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

// NewResponsesHandlerWithRegistryAndGenerations adds optional telemetry persistence to the shared manager.
func NewResponsesHandlerWithRegistryAndGenerations(registry *providers.Registry, manager *execution.Manager, logger *slog.Logger, inactivity time.Duration, routingDeps *ResponsesRouting, resourceAdmission ResourceAdmitter, concurrencyGate ConcurrencyGate, usage *metering.Service, generationService *generations.Service) http.Handler {
	handler := NewResponsesHandlerWithRegistry(registry, manager, logger, inactivity, routingDeps, resourceAdmission, concurrencyGate, usage)
	responses, ok := handler.(*responsesHandler)
	if ok {
		responses.generations = generationService
	}
	return handler
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
	generations *generations.Service
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
	clientApp := clientAppFromRequest(r)
	providerID := providers.OpenCodeGoID
	modelID := requestedModelID(payload)
	executionPayload := payload

	var routingDecision routing.RoutingDecision
	pendingRoutingLog := false
	var routingOverheadNS *int64
	var routingSelectedAt time.Time
	if h.routing != nil {
		var target routing.Target
		var prepared []byte
		var routeErr error
		routingStartedAt := time.Now()
		target, prepared, routingDecision, routeErr = routeResponses(r.Context(), h.routing, h.registry, requestID, explicitProvider, payload)
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
		if routingDecision.Origin == routing.OriginVirtualAuto && target.ModelID != "" {
			w.Header().Set(routing.HeaderResolvedModel, target.ModelID)
			w.Header().Set(routing.HeaderResolvedProvider, target.ProviderID)
			profile := routingDecision.AppliedProfile
			if profile == "" {
				profile = routing.ProfileBalanced
			}
			w.Header().Set(routing.HeaderRoutingProfile, string(profile))
			reason := routingDecision.RoutingReasonCode
			if reason == "" && len(routingDecision.Criteria) > 0 {
				reason = routingDecision.Criteria[0]
			}
			w.Header().Set(routing.HeaderRoutingReason, reason)
		}
		if !h.requireProvider(providerID, w) {
			logRoutingDecision(false)
			return
		}
		elapsed := int64(time.Since(routingStartedAt))
		routingOverheadNS = &elapsed
		routingSelectedAt = time.Now().UTC()
		pendingRoutingLog = true
		if routingDecision.SelectedPool != "" {
			defer routing.ReleaseProvisionalDebit(routing.AutoPoolID(routingDecision.SelectedPool))
		}
	} else {
		providerID, err = requestedProviderID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_request", "the provider selector is invalid")
			return
		}
		if !h.requireProvider(providerID, w) {
			return
		}
		executionPayload, err = adaptResponsesPayloadForProvider(providerID, executionPayload)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_request", "the request body is invalid")
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
	var execSink execution.Sink = responseWriter
	if routingDecision.Origin == routing.OriginVirtualAuto && modelID != "" && modelID != routing.VirtualModelAuto {
		execSink = newVirtualAutoModelSink(responseWriter, modelID)
	}
	result, err := h.manager.Execute(execCtx, execution.Command{
		RequestID:   requestID,
		ExecutionID: executionID,
		Target: execution.Target{
			ProviderID: providerID,
			ModelID:    modelID,
		},
		ProviderSessionID: sessionID,
		RoutingOverheadNS: routingOverheadNS,
		RoutingSelectedAt: routingSelectedAt,
		Payload:           executionPayload,
		ContentType:       contentType,
	}, execSink)
	captured := streamTap.Finalize(responseWriter.contentType)
	if h.usage != nil && err == nil {
		h.usage.RecordFromCaptureBestEffort(r.Context(), metering.RecordInput{
			ExecutionID: result.ExecutionID,
			ProviderID:  providerID,
			ModelID:     modelID,
			Capture:     captured,
		})
	}
	if h.generations != nil && err == nil {
		h.generations.RecordBestEffort(r.Context(), generations.TelemetryInput{
			ExecutionID:  result.ExecutionID,
			ProviderID:   providerID,
			ModelID:      modelID,
			SessionID:    sessionID,
			ClientApp:    clientApp,
			FinishReason: capture.ExtractFinishReason(captured),
			Result:       result,
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

	writeExecutionFailureResponse(w, result, routing.Target{ProviderID: providerID, ModelID: modelID})
}

// writeExecutionFailureResponse maps an uncommitted terminal opening result
// to the structured endpoint error. Upstream quota/subscription exhaustion
// answers 429 with the bounded reset window; everything else keeps the
// historical 502/504 envelopes.
func writeExecutionFailureResponse(w http.ResponseWriter, result execution.Result, routedTarget routing.Target) {
	if result.Outcome == execution.OutcomeTimedOut && result.Phase == execution.PhaseOpening {
		writeJSONError(w, http.StatusGatewayTimeout, executionTimeoutType, executionTimeoutMsg)
		return
	}
	if result.Outcome == execution.OutcomeFailed && result.Phase == execution.PhaseOpening {
		if result.Cause == execution.CauseUpstreamQuota {
			resetAt := ""
			var resetTime time.Time
			if result.QuotaResetInNS != nil && *result.QuotaResetInNS > 0 {
				resetTime = time.Now().Add(time.Duration(*result.QuotaResetInNS)).UTC()
				resetAt = resetTime.Format(time.RFC3339)
			}
			if routedTarget.ProviderID != "" && routedTarget.ModelID != "" {
				routing.RecordAutoQuotaExhausted(routedTarget, resetTime)
			}
			writeJSONErrorWithReset(w, http.StatusTooManyRequests, upstreamQuotaExceededType, upstreamQuotaExceededMsg, resetAt)
			return
		}
		if result.UpstreamStatus == http.StatusTooManyRequests && routedTarget.ProviderID == providers.OpenCodeGoID {
			routing.RecordAutoOverloadCooldown(routedTarget)
		}
		writeJSONError(w, http.StatusBadGateway, upstreamUnavailableType, upstreamUnavailableMsg)
	}
}
