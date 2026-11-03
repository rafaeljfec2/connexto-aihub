package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"aihub/internal/health"
	"aihub/internal/operational"
	"aihub/internal/providers"
)

const operationalHeartbeatInterval = 30 * time.Second

// OperationalEventsHandler serves GET /api/events as a long-lived operational SSE stream.
type OperationalEventsHandler struct {
	hub       *operational.Hub
	readiness *health.Registry
	statuses  func() []providers.Status
	logger    *slog.Logger
}

// NewOperationalEventsHandler builds the operational SSE handler.
func NewOperationalEventsHandler(hub *operational.Hub, readiness *health.Registry, statuses func() []providers.Status, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	handler := &OperationalEventsHandler{
		hub:       hub,
		readiness: readiness,
		statuses:  statuses,
		logger:    logger,
	}

	return http.HandlerFunc(handler.serve)
}

func (h *OperationalEventsHandler) serve(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, internalErrorType, "streaming is unavailable")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	events, unsubscribe := h.hub.Subscribe()
	defer unsubscribe()

	if err := h.writeReadiness(ctx, w, flusher); err != nil {
		h.logger.Warn("operational sse readiness snapshot failed", "error", err.Error())
		return
	}
	if err := h.writeProviderStatus(w, flusher); err != nil {
		h.logger.Warn("operational sse provider snapshot failed", "error", err.Error())
		return
	}

	ticker := time.NewTicker(operationalHeartbeatInterval)
	defer ticker.Stop()

	lastReadiness := h.readiness.Report(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := writeOperationalEvent(w, flusher, operational.EventHeartbeat, map[string]int{
				"schema_version": operational.SchemaVersion,
			}); err != nil {
				return
			}
			report := h.readiness.Report(ctx)
			if readinessReportChanged(lastReadiness, report) {
				lastReadiness = report
				PublishReadiness(h.hub, report)
			}
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeOperationalRawEvent(w, flusher, event.Name, event.Data); err != nil {
				return
			}
		}
	}
}

func (h *OperationalEventsHandler) writeReadiness(ctx context.Context, w http.ResponseWriter, flusher http.Flusher) error {
	report := h.readiness.Report(ctx)
	return writeOperationalEvent(w, flusher, operational.EventReadinessChanged, readinessPayload(report))
}

func (h *OperationalEventsHandler) writeProviderStatus(w http.ResponseWriter, flusher http.Flusher) error {
	return writeOperationalEvent(w, flusher, operational.EventProviderStatus, providerStatusPayload(h.statuses()))
}

func readinessPayload(report health.Report) map[string]any {
	payload := map[string]any{
		"schema_version": operational.SchemaVersion,
		"status":         report.Status,
	}
	if len(report.Checks) > 0 {
		payload["checks"] = report.Checks
	}
	return payload
}

func providerStatusPayload(statuses []providers.Status) map[string]any {
	return map[string]any{
		"schema_version": operational.SchemaVersion,
		"providers":        statuses,
	}
}

func readinessReportChanged(previous, current health.Report) bool {
	if previous.Status != current.Status {
		return true
	}
	if len(previous.Checks) != len(current.Checks) {
		return true
	}
	for name, status := range previous.Checks {
		if current.Checks[name] != status {
			return true
		}
	}
	return false
}

func writeOperationalEvent(w http.ResponseWriter, flusher http.Flusher, name string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return writeOperationalRawEvent(w, flusher, name, data)
}

func writeOperationalRawEvent(w http.ResponseWriter, flusher http.Flusher, name string, data []byte) error {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

// PublishReadiness emits a readiness_changed event to subscribers.
func PublishReadiness(hub *operational.Hub, report health.Report) {
	if hub == nil {
		return
	}
	hub.Publish(operational.EventReadinessChanged, readinessPayload(report))
}

// PublishProviderStatus emits a provider_status event to subscribers.
func PublishProviderStatus(hub *operational.Hub, statuses func() []providers.Status) {
	if hub == nil || statuses == nil {
		return
	}
	hub.Publish(operational.EventProviderStatus, providerStatusPayload(statuses()))
}
