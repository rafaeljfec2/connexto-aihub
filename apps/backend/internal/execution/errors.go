package execution

import (
	"errors"
	"time"
)

var (
	ErrUnavailable        = errors.New("execution unavailable")
	ErrInvalidCommand     = errors.New("invalid execution command")
	ErrCleanupUnconfirmed = errors.New("execution cleanup could not be confirmed")
)

type responseHeaderTimeout interface {
	IsResponseHeaderTimeout() bool
}

// upstreamQuotaNotice mirrors providers.QuotaNoticeError without importing
// the provider package (same seam as responseHeaderTimeout).
type upstreamQuotaNotice interface {
	IsUpstreamQuotaNotice() bool
	QuotaResetIn() time.Duration
}

func quotaNoticeReset(err error) (*int64, bool) {
	var notice upstreamQuotaNotice
	if !errors.As(err, &notice) {
		return nil, false
	}
	return durationValue(notice.QuotaResetIn()), true
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
