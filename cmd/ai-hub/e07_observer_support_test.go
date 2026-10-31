package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"aihub/internal/execution"
)

type e07ReplayObservation struct {
	PriorContextPresent       bool
	PreviousResponseIDPresent bool
}

type e07ProviderObservation struct {
	Calls  uint64
	Replay e07ReplayObservation
}

type e07ProviderObserver struct {
	base   http.RoundTripper
	marker string

	inspectReplay atomic.Bool
	calls         atomic.Uint64
	replayMu      sync.Mutex
	replay        e07ReplayObservation
}

func newE07ProviderObserver(base http.RoundTripper, marker string) *e07ProviderObserver {
	return &e07ProviderObserver{base: base, marker: marker}
}

func (o *e07ProviderObserver) RoundTrip(request *http.Request) (*http.Response, error) {
	o.calls.Add(1)
	if o.inspectReplay.Load() && o.marker != "" && request.Body != nil {
		payload, err := io.ReadAll(request.Body)
		_ = request.Body.Close()
		if err != nil {
			return nil, err
		}
		request.Body = io.NopCloser(bytes.NewReader(payload))

		o.replayMu.Lock()
		if bytes.Contains(payload, []byte(o.marker)) {
			o.replay.PriorContextPresent = true
		}
		if bytes.Contains(payload, []byte("previous_response_id")) {
			o.replay.PreviousResponseIDPresent = true
		}
		o.replayMu.Unlock()
	}

	return o.base.RoundTrip(request)
}

func (o *e07ProviderObserver) Snapshot() e07ProviderObservation {
	o.replayMu.Lock()
	defer o.replayMu.Unlock()

	return e07ProviderObservation{Calls: o.calls.Load(), Replay: o.replay}
}

// e07TerminalObservation is the validation-only copy of one E05 terminal
// result. It retains no payload, provider session, or response content.
type e07TerminalObservation struct {
	RequestID   string
	ExecutionID string
	ProviderID  string
	Outcome     string
}

// e07TerminalRecorder records E05 terminal outcomes in memory and delegates to
// the real E06 observer. It is never part of production composition.
type e07TerminalRecorder struct {
	next execution.ResultObserver

	mu           sync.Mutex
	observations []e07TerminalObservation
}

func newE07TerminalRecorder(next execution.ResultObserver) *e07TerminalRecorder {
	return &e07TerminalRecorder{next: next}
}

func (r *e07TerminalRecorder) OnExecutionFinished(finished execution.FinishedExecution) {
	r.mu.Lock()
	r.observations = append(r.observations, e07TerminalObservation{
		RequestID:   finished.RequestID,
		ExecutionID: finished.Result.ExecutionID,
		ProviderID:  finished.Target.ProviderID,
		Outcome:     string(finished.Result.Outcome),
	})
	r.mu.Unlock()

	if r.next != nil {
		r.next.OnExecutionFinished(finished)
	}
}

func (r *e07TerminalRecorder) Snapshot() []e07TerminalObservation {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]e07TerminalObservation(nil), r.observations...)
}

func e07CodexEnvironment(parentEnvironment []string, codexHome, gatewayKey string) []string {
	parent := make(map[string]string, len(parentEnvironment))
	for _, entry := range parentEnvironment {
		key, value, found := strings.Cut(entry, "=")
		if found {
			parent[key] = value
		}
	}

	allowed := []string{"PATH", "USER", "LOGNAME", "LANG", "LC_ALL", "TERM", "TMPDIR", "TEMP", "TMP", "SYSTEMROOT", "WINDIR"}
	environment := make([]string, 0, len(allowed)+2)
	for _, key := range allowed {
		if value, exists := parent[key]; exists {
			environment = append(environment, key+"="+value)
		}
	}
	environment = append(environment, "HOME="+codexHome, "CODEX_HOME="+codexHome, "AIHUB_CLIENT_KEY="+gatewayKey)
	return environment
}
