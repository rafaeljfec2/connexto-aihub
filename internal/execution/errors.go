package execution

import "errors"

var (
	ErrUnavailable        = errors.New("execution unavailable")
	ErrInvalidCommand     = errors.New("invalid execution command")
	ErrCleanupUnconfirmed = errors.New("execution cleanup could not be confirmed")
)

type responseHeaderTimeout interface {
	IsResponseHeaderTimeout() bool
}

type terminalEvent struct {
	outcome Outcome
	phase   Phase
	cause   Cause
}

func (event terminalEvent) finalState() state {
	switch event.outcome {
	case OutcomeCompleted:
		return stateCompleted
	case OutcomeFailed:
		return stateFailed
	case OutcomeCanceled:
		return stateCanceled
	default:
		return stateTimedOut
	}
}
