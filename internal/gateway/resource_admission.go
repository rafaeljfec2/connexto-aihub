package gateway

import (
	"context"

	"aihub/internal/resources"
)

// ResourceAdmitter performs E09 admission after routing and before execution.
type ResourceAdmitter interface {
	Admit(ctx context.Context, request resources.AdmissionRequest) (resources.AdmissionDecision, error)
}
