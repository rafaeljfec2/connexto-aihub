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

	e.mu.Lock()
	e.state = event.finalState()
	result := Result{
		ExecutionID:    e.id,
		Outcome:        event.outcome,
		Phase:          event.phase,
		Cause:          event.cause,
		UpstreamStatus: e.status,
		Committed:      e.committed,
		Duration:       time.Since(e.started),
	}
	e.mu.Unlock()
	e.manager.logFinished(result, e.command)
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
