package records

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

// SchemaVersion is the only record schema understood by this build.
const SchemaVersion = 1

const (
	executionIDBytes   = 32
	minRequestIDBytes  = 8
	maxRequestIDBytes  = 128
	maxProviderIDBytes = 128
	maxModelIDBytes    = 4096
	maxRecordLineBytes = 16 << 10
	minStatus          = 100
	maxStatus          = 599
)

// Terminal outcome vocabulary copied from the execution boundary.
const (
	OutcomeCompleted = "completed"
	OutcomeFailed    = "failed"
	OutcomeCanceled  = "canceled"
	OutcomeTimedOut  = "timed_out"
)

// Execution phase vocabulary copied from the execution boundary.
const (
	PhaseOpening    = "opening"
	PhaseForwarding = "forwarding"
)

// Cause vocabulary copied from the execution boundary.
const (
	CauseTransport    = "transport"
	CauseUpstreamRead = "upstream_read"
	CauseClient       = "client"
	CauseShutdown     = "shutdown"
	CauseOpening      = "opening"
	CauseInactivity   = "inactivity"
)

// ErrInvalidRecord marks values outside the schema-v1 contract.
var ErrInvalidRecord = errors.New("invalid record")

// Record is the immutable schema-v1 execution record.
type Record struct {
	SchemaVersion     int
	ExecutionID       string
	RequestID         *string
	ProviderID        string
	ModelID           *string
	StartedAt         time.Time
	CompletedAt       time.Time
	DurationNS        int64
	Outcome           string
	Phase             string
	Cause             *string
	UpstreamStatus    *int
	ResponseCommitted bool
}

// Validate reports whether the record satisfies every schema-v1 bound.
func (r Record) Validate() error {
	if r.SchemaVersion != SchemaVersion {
		return invalidRecord("unsupported schema_version %d", r.SchemaVersion)
	}
	if !validExecutionID(r.ExecutionID) {
		return invalidRecord("execution_id must be 32 lowercase hexadecimal characters")
	}
	if r.RequestID != nil && !validRequestID(*r.RequestID) {
		return invalidRecord("request_id is outside the correlation grammar")
	}
	if !validUTF8Value(r.ProviderID, 1, maxProviderIDBytes) {
		return invalidRecord("provider_id is outside its byte bounds")
	}
	if r.ModelID != nil && !validUTF8Value(*r.ModelID, 1, maxModelIDBytes) {
		return invalidRecord("model_id is outside its byte bounds")
	}
	if !validTimestamp(r.StartedAt) {
		return invalidRecord("started_at must be a non-zero UTC timestamp")
	}
	if !validTimestamp(r.CompletedAt) {
		return invalidRecord("completed_at must be a non-zero UTC timestamp")
	}
	if r.DurationNS < 0 {
		return invalidRecord("duration_ns must be non-negative")
	}
	if !r.CompletedAt.Equal(r.StartedAt.Add(time.Duration(r.DurationNS))) {
		return invalidRecord("completed_at must equal started_at plus duration_ns")
	}
	if !knownOutcome(r.Outcome) {
		return invalidRecord("unsupported outcome %q", r.Outcome)
	}
	if !knownPhase(r.Phase) {
		return invalidRecord("unsupported phase %q", r.Phase)
	}
	if r.Outcome == OutcomeCompleted {
		if r.Cause != nil {
			return invalidRecord("cause must be null for a completed execution")
		}
	} else if r.Cause == nil || !knownCause(*r.Cause) {
		return invalidRecord("non-completed executions require a known cause")
	}
	if r.UpstreamStatus != nil && (*r.UpstreamStatus < minStatus || *r.UpstreamStatus > maxStatus) {
		return invalidRecord("upstream_status must be between 100 and 599")
	}

	return nil
}

func invalidRecord(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRecord, fmt.Sprintf(format, args...))
}

// ValidExecutionID reports whether value matches the schema-v1 identifier.
func ValidExecutionID(value string) bool {
	return validExecutionID(value)
}

func validExecutionID(value string) bool {
	if len(value) != executionIDBytes {
		return false
	}

	for _, char := range value {
		if char < '0' || (char > '9' && char < 'a') || char > 'f' {
			return false
		}
	}

	return true
}

func validRequestID(value string) bool {
	if len(value) < minRequestIDBytes || len(value) > maxRequestIDBytes {
		return false
	}

	for index := 0; index < len(value); index++ {
		char := value[index]
		allowed := char >= 'A' && char <= 'Z' ||
			char >= 'a' && char <= 'z' ||
			char >= '0' && char <= '9' ||
			char == '.' || char == '_' || char == ':' || char == '-'
		if !allowed {
			return false
		}
	}

	return true
}

func validUTF8Value(value string, minimum, maximum int) bool {
	if len(value) < minimum || len(value) > maximum || !utf8.ValidString(value) {
		return false
	}

	return true
}

func validTimestamp(value time.Time) bool {
	return !value.IsZero() && value.Location() == time.UTC
}

func knownOutcome(value string) bool {
	switch value {
	case OutcomeCompleted, OutcomeFailed, OutcomeCanceled, OutcomeTimedOut:
		return true
	default:
		return false
	}
}

func knownPhase(value string) bool {
	return value == PhaseOpening || value == PhaseForwarding
}

func knownCause(value string) bool {
	switch value {
	case CauseTransport, CauseUpstreamRead, CauseClient, CauseShutdown, CauseOpening, CauseInactivity:
		return true
	default:
		return false
	}
}
