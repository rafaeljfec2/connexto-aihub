package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aihub/internal/access"
	"aihub/internal/execution"
	"aihub/internal/observability"
	"aihub/internal/providers"
)

// StreamInactivityTimeout bounds silence between streamed chunks.
const StreamInactivityTimeout = 2 * time.Minute

const (
	sessionHeader = "X-Opencode-Session"
	SessionPrefix = "aihub-ses-"
	// Keep captured identifiers within the execution-record model_id bound.
	maxObservedModelIDBytes = 4096
)

// NewResponsesHandler creates a standalone execution manager for this handler.
func NewResponsesHandler(client *providers.Client, logger *slog.Logger, inactivity time.Duration) http.Handler {
	manager := newExecutionManager(client, logger, inactivity)
	return NewResponsesHandlerWithManager(client, manager, logger, inactivity)
}

// NewResponsesHandlerWithManager binds the HTTP boundary to the shared manager.
func NewResponsesHandlerWithManager(client *providers.Client, manager *execution.Manager, logger *slog.Logger, inactivity time.Duration) http.Handler {
	if inactivity <= 0 {
		inactivity = StreamInactivityTimeout
	}
	return &responsesHandler{client: client, manager: manager, logger: logger, inactivity: inactivity}
}

// NewResponsesHandlerWithRegistry resolves each admitted request by provider ID.
func NewResponsesHandlerWithRegistry(registry *providers.Registry, manager *execution.Manager, logger *slog.Logger, inactivity time.Duration) http.Handler {
	if inactivity <= 0 {
		inactivity = StreamInactivityTimeout
	}
	return &responsesHandler{registry: registry, manager: manager, logger: logger, inactivity: inactivity}
}

type responsesHandler struct {
	client     *providers.Client
	registry   *providers.Registry
	manager    *execution.Manager
	logger     *slog.Logger
	inactivity time.Duration
}

func (h *responsesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	providerID, err := requestedProviderID(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "the provider selector is invalid")
		return
	}
	if !h.requireProvider(providerID, w) {
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

	result, err := h.manager.Execute(r.Context(), execution.Command{
		RequestID: observability.CorrelationIDFrom(r.Context()),
		Target: execution.Target{
			ProviderID: providerID,
			ModelID:    requestedModelID(payload),
		},
		ProviderSessionID: sessionID,
		Payload:           payload,
		ContentType:       contentType,
	}, &responseSink{writer: w, inactivity: h.inactivity})
	if errors.Is(err, execution.ErrUnavailable) {
		writeJSONError(w, http.StatusServiceUnavailable, executionUnavailableType, executionUnavailableMsg)
		return
	}
	if err != nil {
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

type responseSink struct {
	writer     http.ResponseWriter
	inactivity time.Duration
}

func (s *responseSink) Commit(status int, contentType string) {
	if contentType != "" {
		s.writer.Header().Set("Content-Type", contentType)
	}
	controller := http.NewResponseController(s.writer)
	_ = controller.SetWriteDeadline(time.Now().Add(s.inactivity))
	s.writer.WriteHeader(status)
}

func (s *responseSink) WriteChunk(chunk []byte) error {
	if _, err := s.writer.Write(chunk); err != nil {
		return err
	}
	controller := http.NewResponseController(s.writer)
	if err := controller.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return err
	}
	_ = controller.SetWriteDeadline(time.Now().Add(s.inactivity))
	return nil
}

func newExecutionManager(client *providers.Client, logger *slog.Logger, inactivity time.Duration) *execution.Manager {
	return execution.NewManager(logger, providers.OpenCodeGoID,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return client.OpenTargetResponses(ctx, providers.ResponsesTarget{
				ProviderID: target.ProviderID,
				ModelID:    target.ModelID,
			}, payload, contentType, sessionID)
		}, inactivity)
}

func providerSessionID(r *http.Request) (string, error) {
	if metadata, ok := access.MetadataFrom(r.Context()); ok && metadata.ProviderSessionID != "" {
		return metadata.ProviderSessionID, nil
	}
	return resolveSessionID(r)
}

func resolveSessionID(r *http.Request) (string, error) {
	if supplied := r.Header.Get(sessionHeader); supplied != "" {
		return supplied, nil
	}
	return newSessionID()
}

func newSessionID() (string, error) {
	return newPrefixedID(SessionPrefix)
}

func newPrefixedID(prefix string) (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate provider session id: %w", err)
	}
	return prefix + hex.EncodeToString(random[:]), nil
}
