package metering

import "errors"

// ErrNotFound indicates that no usage row exists for the execution.
var ErrNotFound = errors.New("metering: not found")
