package execution

import (
	"errors"
	"io"
	"time"
)

func (e *execution) run(sink Sink) Result {
	e.setState(stateOpening)
	e.dispatchProvider()
	response, openErr := e.manager.opener(e.ctx, e.command.Target, e.command.Payload,
		e.command.ContentType, e.command.ProviderSessionID)
	e.observeProviderResponse(response, time.Now())
	if response != nil && response.Body != nil {
		e.attachBody(response.Body)
	}
	if openErr != nil {
		e.reserveOpenFailure(openErr)
		return e.finish()
	}
	if response == nil || response.Body == nil {
		e.reserve(terminalEvent{outcome: OutcomeFailed, phase: PhaseOpening, cause: CauseTransport})
		return e.finish()
	}

	e.mu.Lock()
	e.status = response.StatusCode
	if e.reserved {
		e.mu.Unlock()
		return e.finish()
	}

	// Serialize metadata commitment with terminal reservation so shutdown
	// cannot win and then have the Gateway commit headers afterward.
	sink.Commit(response.StatusCode, response.Header.Get("Content-Type"))
	e.committed = true
	e.state = stateForwarding
	e.mu.Unlock()

	e.forward(sink)
	return e.finish()
}

func (e *execution) reserveOpenFailure(openErr error) {
	e.mu.Lock()
	reserved := e.reserved
	e.mu.Unlock()
	if reserved {
		return
	}
	if e.ctx.Err() != nil {
		e.requestCancellation(terminalEvent{outcome: OutcomeCanceled, phase: PhaseOpening, cause: CauseClient})
		return
	}
	if isHeaderTimeout(openErr) {
		e.reserve(terminalEvent{outcome: OutcomeTimedOut, phase: PhaseOpening, cause: CauseOpening})
		return
	}
	if quotaReset, ok := quotaNoticeReset(openErr); ok {
		e.quotaResetIn = quotaReset
		e.reserve(terminalEvent{outcome: OutcomeFailed, phase: PhaseOpening, cause: CauseUpstreamQuota})
		return
	}
	e.reserve(terminalEvent{outcome: OutcomeFailed, phase: PhaseOpening, cause: CauseTransport})
}

func (e *execution) forward(sink Sink) {
	chunks := e.startReader()
	timer := time.NewTimer(e.manager.inactivity)
	defer timer.Stop()

	for {
		if e.hasReserved() {
			return
		}
		select {
		case <-e.ctx.Done():
			e.requestCancellation(terminalEvent{outcome: OutcomeCanceled, phase: PhaseForwarding, cause: CauseClient})
			return
		case <-timer.C:
			e.reserve(terminalEvent{outcome: OutcomeTimedOut, phase: PhaseForwarding, cause: CauseInactivity})
			return
		case chunk := <-chunks:
			if e.hasReserved() {
				return
			}
			if chunk.err != nil {
				if errors.Is(chunk.err, io.EOF) {
					e.reserve(terminalEvent{outcome: OutcomeCompleted, phase: PhaseForwarding})
				} else {
					e.reserve(terminalEvent{outcome: OutcomeFailed, phase: PhaseForwarding, cause: CauseUpstreamRead})
				}
				return
			}
			if err := sink.WriteChunk(chunk.data); err != nil {
				e.requestCancellation(terminalEvent{outcome: OutcomeCanceled, phase: PhaseForwarding, cause: CauseClient})
				return
			}
			e.observeFirstOutput(time.Now())
			rearm(timer, e.manager.inactivity)
		}
	}
}
func (e *execution) startReader() <-chan readResult {
	chunks := make(chan readResult, 1)
	done := make(chan struct{})
	e.mu.Lock()
	e.readerDone = done
	e.mu.Unlock()
	go func() {
		defer close(done)
		buffer := make([]byte, 32<<10)
		for {
			select {
			case <-e.stopReader:
				return
			default:
			}
			if e.hasReserved() {
				return
			}
			n, err := e.currentBody().Read(buffer)
			if n > 0 {
				data := append([]byte(nil), buffer[:n]...)
				select {
				case chunks <- readResult{data: data}:
				case <-e.stopReader:
					return
				}
			}
			if err != nil {
				select {
				case chunks <- readResult{err: err}:
				case <-e.stopReader:
				}
				return
			}
		}
	}()
	return chunks
}

func rearm(timer *time.Timer, duration time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(duration)
}
