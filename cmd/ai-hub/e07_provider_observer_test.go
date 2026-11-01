package main

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func e07ProviderHTTPClient(observer *e07ProviderObserver, timeout time.Duration) *http.Client {
	return &http.Client{Transport: observer, Timeout: timeout}
}

type e07ProviderObserver struct {
	base   http.RoundTripper
	marker string
	budget *e07ProviderBudget

	inspectReplay atomic.Bool
	calls         atomic.Uint64
	replayMu      sync.Mutex
	replay        e07ReplayObservation
	responseMu    sync.Mutex
	captureNext   bool
	response      *e07ResponseFidelityObservation
}

func newE07ProviderObserver(base http.RoundTripper, marker string) *e07ProviderObserver {
	budget, err := newE07ProviderBudget(e07ProviderCallLimit)
	if err != nil {
		panic("approved E07 provider-call budget is invalid")
	}
	return newE07ProviderObserverWithBudget(base, marker, budget)
}

func newE07ProviderObserverWithBudget(base http.RoundTripper, marker string, budget *e07ProviderBudget) *e07ProviderObserver {
	if base == nil {
		base = http.DefaultTransport
	}
	if budget == nil {
		panic("E07 provider observer requires a shared call budget")
	}
	return &e07ProviderObserver{base: base, marker: marker, budget: budget}
}

func (observer *e07ProviderObserver) RoundTrip(request *http.Request) (*http.Response, error) {
	if !observer.budget.reserve() {
		return nil, errE07ProviderBudgetExhausted
	}
	if observer.inspectReplay.Load() && observer.marker != "" && request.Body != nil {
		payload, err := io.ReadAll(request.Body)
		_ = request.Body.Close()
		if err != nil {
			return nil, err
		}
		request.Body = io.NopCloser(bytes.NewReader(payload))

		observer.replayMu.Lock()
		if bytes.Contains(payload, []byte(observer.marker)) {
			observer.replay.PriorContextPresent = true
		}
		if bytes.Contains(payload, []byte("previous_response_id")) {
			observer.replay.PreviousResponseIDPresent = true
		}
		observer.replayMu.Unlock()
	}

	observer.budget.markForwarded()
	observer.calls.Add(1)
	response, err := observer.base.RoundTrip(request)
	observer.observeResponse(response, err)
	return response, err
}

func (observer *e07ProviderObserver) BudgetSnapshot() e07ProviderBudgetSnapshot {
	return observer.budget.Snapshot()
}

func (observer *e07ProviderObserver) CloseIdleConnections() {
	if closer, ok := observer.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (observer *e07ProviderObserver) armResponseFidelityObservation() {
	observer.responseMu.Lock()
	observer.captureNext = true
	observer.response = nil
	observer.responseMu.Unlock()
}

func (observer *e07ProviderObserver) takeResponseFidelityObservation() e07ResponseFidelityObservation {
	observer.responseMu.Lock()
	defer observer.responseMu.Unlock()

	observer.captureNext = false
	if observer.response == nil {
		return e07ResponseFidelityObservation{}
	}
	observation := *observer.response
	observer.response = nil
	return observation
}

func (observer *e07ProviderObserver) observeResponse(response *http.Response, responseErr error) {
	observer.responseMu.Lock()
	if !observer.captureNext {
		observer.responseMu.Unlock()
		return
	}
	observer.captureNext = false
	observation := &e07ResponseFidelityObservation{}
	if response != nil {
		observation.StatusCode = response.StatusCode
		observation.ContentType = response.Header.Get("Content-Type")
	}
	observer.response = observation
	if responseErr != nil || response == nil || response.Body == nil {
		observer.responseMu.Unlock()
		return
	}
	response.Body = newE07DigestReadCloser(response.Body, func(digest [32]byte, complete bool) {
		observer.responseMu.Lock()
		observation.BodyDigest = digest
		observation.BodyComplete = complete
		observer.responseMu.Unlock()
	})
	observer.responseMu.Unlock()
}

func (observer *e07ProviderObserver) Snapshot() e07ProviderObservation {
	observer.replayMu.Lock()
	defer observer.replayMu.Unlock()

	return e07ProviderObservation{Calls: observer.calls.Load(), Replay: observer.replay}
}
