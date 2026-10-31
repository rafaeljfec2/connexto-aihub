package execution

import (
	"context"
	"net/http"
	"time"
)

// Target identifies the provider destination already selected by the caller.
type Target struct {
	ProviderID string
	ModelID    string
}

// Command contains the accepted provider-ready request and its execution context.
type Command struct {
	RequestID         string
	Target            Target
	ProviderSessionID string
	Payload           []byte
	ContentType       string
}

// Outcome is the immutable terminal classification of an execution.
type Outcome string

const (
	OutcomeCompleted Outcome = "completed"
	OutcomeFailed    Outcome = "failed"
	OutcomeCanceled  Outcome = "canceled"
	OutcomeTimedOut  Outcome = "timed_out"
)

// Phase identifies where an execution reached its terminal event.
type Phase string

const (
	PhaseOpening    Phase = "opening"
	PhaseForwarding Phase = "forwarding"
)

// Cause records the reason for a non-success terminal outcome.
type Cause string

const (
	CauseTransport    Cause = "transport"
	CauseUpstreamRead Cause = "upstream_read"
	CauseClient       Cause = "client"
	CauseShutdown     Cause = "shutdown"
	CauseOpening      Cause = "opening"
	CauseInactivity   Cause = "inactivity"
)

// Result is published after the upstream body is closed and the reader joins.
type Result struct {
	ExecutionID    string
	Outcome        Outcome
	Phase          Phase
	Cause          Cause
	UpstreamStatus int
	Committed      bool
	Duration       time.Duration
}

// FinishedExecution is the minimal immutable observation offered to consumers
// after an execution reaches its terminal result. It carries no provider
// session, payload, response content, or HTTP headers.
type FinishedExecution struct {
	Result    Result
	RequestID string
	Target    Target
	StartedAt time.Time
}

// ResultObserver receives one finished execution per registered execution.
// Implementations must return promptly and must not mutate the value.
type ResultObserver interface {
	OnExecutionFinished(FinishedExecution)
}

// Sink keeps HTTP response ownership in the gateway.
type Sink interface {
	Commit(status int, contentType string)
	WriteChunk([]byte) error
}

// OpenResponses is the narrow test seam for the configured concrete adapter.
type OpenResponses func(context.Context, Target, []byte, string, string) (*http.Response, error)

type state string

const (
	stateAccepted   state = "accepted"
	stateOpening    state = "opening"
	stateForwarding state = "forwarding"
	stateCompleted  state = "completed"
	stateFailed     state = "failed"
	stateCanceled   state = "canceled"
	stateTimedOut   state = "timed_out"
)
