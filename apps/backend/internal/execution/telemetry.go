package execution

import (
	"net/http"
	"strings"
	"time"
)

// EventKind identifies one observed, content-free generation lifecycle event.
type EventKind string

const (
	EventRoutingSelected         EventKind = "routing_selected"
	EventUpstreamRequestStarted  EventKind = "upstream_request_started"
	EventUpstreamResponseArrived EventKind = "upstream_response_received"
	EventFirstOutput             EventKind = "first_output"
	EventGenerationFinished      EventKind = "generation_finished"
)

// TelemetryEvent contains only bounded metadata observed by the gateway.
type TelemetryEvent struct {
	Kind           EventKind
	ObservedAt     time.Time
	ProviderID     string
	ModelID        string
	UpstreamStatus *int
	DurationNS     *int64
}

func durationValue(duration time.Duration) *int64 {
	if duration < 0 {
		return nil
	}
	value := int64(duration)
	return &value
}

func (e *execution) dispatchProvider() time.Time {
	e.dispatchedAt = time.Now()
	e.addTelemetryEvent(TelemetryEvent{
		Kind:       EventUpstreamRequestStarted,
		ObservedAt: e.dispatchedAt.UTC(),
		ProviderID: e.command.Target.ProviderID,
		ModelID:    e.command.Target.ModelID,
	})

	return e.dispatchedAt
}

func (e *execution) observeProviderResponse(response *http.Response, observedAt time.Time) {
	if e.dispatchedAt.IsZero() {
		return
	}
	e.providerLatencyNS = durationValue(observedAt.Sub(e.dispatchedAt))
	if response == nil {
		return
	}

	streaming := strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream")
	e.streaming = &streaming
	status := response.StatusCode
	e.addTelemetryEvent(TelemetryEvent{
		Kind:           EventUpstreamResponseArrived,
		ObservedAt:     observedAt.UTC(),
		ProviderID:     e.command.Target.ProviderID,
		ModelID:        e.command.Target.ModelID,
		UpstreamStatus: &status,
		DurationNS:     e.providerLatencyNS,
	})
}

func (e *execution) observeFirstOutput(observedAt time.Time) {
	if !e.firstOutputAt.IsZero() || e.dispatchedAt.IsZero() {
		return
	}
	e.firstOutputAt = observedAt
	firstOutputLatency := durationValue(observedAt.Sub(e.dispatchedAt))
	e.timeToFirstTokenNS = firstOutputLatency
	e.addTelemetryEvent(TelemetryEvent{
		Kind:       EventFirstOutput,
		ObservedAt: observedAt.UTC(),
		ProviderID: e.command.Target.ProviderID,
		ModelID:    e.command.Target.ModelID,
		DurationNS: firstOutputLatency,
	})
}

func (e *execution) addTelemetryEvent(event TelemetryEvent) {
	e.telemetryEvents = append(e.telemetryEvents, event)
}
