package analytics

import "errors"

var (
	// ErrInvalidInput marks client-visible validation failures.
	ErrInvalidInput = errors.New("invalid analytics input")
)
