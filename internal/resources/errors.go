package resources

import "errors"

var (
	// ErrNotFound means the policy id does not exist.
	ErrNotFound = errors.New("resource policy not found")
	// ErrConflict means the scope key already has a policy.
	ErrConflict = errors.New("resource policy already exists for scope")
	// ErrInvalidInput means caller input failed validation.
	ErrInvalidInput = errors.New("invalid resource input")
	// ErrLimitExceeded means an applicable enabled policy has no capacity.
	ErrLimitExceeded = errors.New("resource limit exceeded")
	// ErrUnavailable means admission could not be decided safely.
	ErrUnavailable = errors.New("resource admission unavailable")
)

// RejectionReasonLimitExceeded is persisted and returned on quota rejections.
const RejectionReasonLimitExceeded = "resource_limit_exceeded"
