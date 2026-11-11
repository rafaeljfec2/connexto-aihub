package execution

import (
	"context"
	"errors"
	"time"
)

func (e *execution) finish() Result {
	e.mu.Lock()
	if !e.reserved {
		e.reserved = true
		e.event = terminalEvent{outcome: OutcomeFailed, phase: PhaseForwarding, cause: CauseUpstreamRead}
	}
	event := e.event
	e.mu.Unlock()
	e.stopOnce.Do(func() { close(e.stopReader) })
	e.cancel(contextCause(event))

	if err := e.closeBody(); err != nil {
		e.manager.recordCleanupFailure()
	}
	e.mu.Lock()
	readerDone := e.readerDone
	e.mu.Unlock()
	if readerDone != nil {
		<-readerDone
	}

	finishedAt := time.Now()
	e.mu.Lock()
	e.state = event.finalState()
	duration := finishedAt.Sub(e.started)
	if !e.firstOutputAt.IsZero() {
		e.outputStreamDurationNS = durationValue(finishedAt.Sub(e.firstOutputAt))
	}
	e.telemetryEvents = append(e.telemetryEvents, TelemetryEvent{
		Kind:       EventGenerationFinished,
		ObservedAt: finishedAt.UTC(),
		ProviderID: e.command.Target.ProviderID,
		ModelID:    e.command.Target.ModelID,
		DurationNS: durationValue(duration),
	})
	result := Result{
		ExecutionID:            e.id,
		Outcome:                event.outcome,
		Phase:                  event.phase,
		Cause:                  event.cause,
		UpstreamStatus:         e.status,
		Committed:              e.committed,
		Duration:               duration,
		Streaming:              e.streaming,
		RoutingOverheadNS:      e.routingOverheadNS,
		ProviderLatencyNS:      e.providerLatencyNS,
		TimeToFirstTokenNS:     e.timeToFirstTokenNS,
		OutputStreamDurationNS: e.outputStreamDurationNS,
		TelemetryEvents:        append([]TelemetryEvent(nil), e.telemetryEvents...),
		QuotaResetInNS:         e.quotaResetIn,
	}
	e.mu.Unlock()
	e.manager.logFinished(result, e.command)
	e.manager.observe(e, result)
	e.manager.remove(e)
	return result
}

func (e *execution) reserve(event terminalEvent) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.reserved {
		return false
	}
	e.reserved = true
	e.event = event
	e.stopOnce.Do(func() { close(e.stopReader) })
	return true
}

func (e *execution) requestCancellation(event terminalEvent) {
	if !e.reserve(event) {
		return
	}
	e.cancel(contextCause(event))
	if err := e.closeBody(); err != nil {
		e.manager.recordCleanupFailure()
	}
}

func (e *execution) cancelForShutdown() {
	e.requestCancellation(terminalEvent{outcome: OutcomeCanceled, phase: e.currentPhase(), cause: CauseShutdown})
}

func (e *execution) currentPhase() Phase {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == stateForwarding {
		return PhaseForwarding
	}
	return PhaseOpening
}

func (e *execution) hasReserved() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.reserved
}

func (e *execution) setState(next state) {
	e.mu.Lock()
	e.state = next
	e.mu.Unlock()
}

func contextCause(event terminalEvent) error {
	if event.cause == "" {
		return context.Canceled
	}
	return errors.New("execution " + string(event.cause))
}
