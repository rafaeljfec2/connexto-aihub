package support

import (
	"strings"
	"sync"

	"aihub/internal/execution"
)

type E07ReplayObservation struct {
	PriorContextPresent       bool
	PreviousResponseIDPresent bool
}

type E07ProviderObservation struct {
	Calls  uint64
	Replay E07ReplayObservation
}

// e07TerminalObservation is the validation-only copy of one E05 terminal
// result. It retains no payload, provider session, or response content.
type E07TerminalObservation struct {
	RequestID   string
	ExecutionID string
	ProviderID  string
	Outcome     string
}

// e07TerminalRecorder records E05 terminal outcomes in memory and delegates to
// the real E06 observer. It is never part of production composition.
type E07TerminalRecorder struct {
	Next execution.ResultObserver

	Mu           sync.Mutex
	Observations []E07TerminalObservation
}

func NewE07TerminalRecorder(next execution.ResultObserver) *E07TerminalRecorder {
	return &E07TerminalRecorder{Next: next}
}

func (r *E07TerminalRecorder) OnExecutionFinished(finished execution.FinishedExecution) {
	r.Mu.Lock()
	r.Observations = append(r.Observations, E07TerminalObservation{
		RequestID:   finished.RequestID,
		ExecutionID: finished.Result.ExecutionID,
		ProviderID:  finished.Target.ProviderID,
		Outcome:     string(finished.Result.Outcome),
	})
	r.Mu.Unlock()

	if r.Next != nil {
		r.Next.OnExecutionFinished(finished)
	}
}

func (r *E07TerminalRecorder) Snapshot() []E07TerminalObservation {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	return append([]E07TerminalObservation(nil), r.Observations...)
}

func E07CodexEnvironment(parentEnvironment []string, codexHome, gatewayKey string) []string {
	parent := make(map[string]string, len(parentEnvironment))
	for _, entry := range parentEnvironment {
		key, value, found := strings.Cut(entry, "=")
		if found {
			parent[key] = value
		}
	}

	allowed := []string{"PATH", "LANG", "LC_ALL", "SYSTEMROOT", "WINDIR"}
	environment := make([]string, 0, len(allowed)+len(E07CodexRuntimeEnvironment(codexHome))+1)
	for _, key := range allowed {
		if value, exists := parent[key]; exists {
			environment = append(environment, key+"="+value)
		}
	}
	environment = append(environment, E07CodexRuntimeEnvironment(codexHome)...)
	environment = append(environment, "AIHUB_CLIENT_KEY="+gatewayKey)
	return environment
}
