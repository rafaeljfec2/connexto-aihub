package concurrency

import "errors"

var (
	ErrNotFound      = errors.New("concurrency policy not found")
	ErrConflict      = errors.New("concurrency policy already exists for scope")
	ErrInvalidInput  = errors.New("invalid concurrency input")
	ErrLimitExceeded = errors.New("concurrency limit exceeded")
	ErrUnavailable   = errors.New("concurrency admission unavailable")
)

const RejectionReasonLimitExceeded = "concurrency_limit_exceeded"

func IsLimitExceeded(err error) bool {
	return errors.Is(err, ErrLimitExceeded)
}
