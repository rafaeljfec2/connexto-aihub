package support

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const E07ProviderCallLimit = 9

var ErrE07ProviderBudgetExhausted = errors.New("E07 provider-call budget exhausted")

type E07ProviderBudgetSnapshot struct {
	Limit     int
	Attempted int
	Admitted  int
	Rejected  int
	Forwarded int
}

type E07ProviderBudget struct {
	Mu    sync.Mutex
	state E07ProviderBudgetSnapshot
}

func NewE07ProviderBudget(limit int) (*E07ProviderBudget, error) {
	if limit < 1 || limit > E07ProviderCallLimit {
		return nil, errors.New("E07 provider-call budget is outside the approved range")
	}
	return &E07ProviderBudget{state: E07ProviderBudgetSnapshot{Limit: limit}}, nil
}

func (budget *E07ProviderBudget) Reserve() bool {
	budget.Mu.Lock()
	defer budget.Mu.Unlock()

	budget.state.Attempted++
	if budget.state.Admitted >= budget.state.Limit {
		budget.state.Rejected++
		return false
	}
	budget.state.Admitted++
	return true
}

func (budget *E07ProviderBudget) MarkForwarded() {
	budget.Mu.Lock()
	budget.state.Forwarded++
	budget.Mu.Unlock()
}

func (budget *E07ProviderBudget) Snapshot() E07ProviderBudgetSnapshot {
	budget.Mu.Lock()
	defer budget.Mu.Unlock()
	return budget.state
}

func NewE07ProviderTransport() (*http.Transport, error) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("default HTTP transport cannot be safely isolated")
	}
	transport := defaultTransport.Clone()
	transport.DisableKeepAlives = true
	return transport, nil
}

func E07ProviderHTTPClient(observer *E07ProviderObserver, timeout time.Duration) *http.Client {
	return &http.Client{Transport: observer, Timeout: timeout}
}

type E07ProviderObserver struct {
	Base   http.RoundTripper
	Marker string
	Budget *E07ProviderBudget

	InspectReplay atomic.Bool
	Calls         atomic.Uint64
	ReplayMu      sync.Mutex
	Replay        E07ReplayObservation
	ResponseMu    sync.Mutex
	CaptureNext   bool
	Response      *E07ResponseFidelityObservation
	Integrity     *E07LiveIntegrityGuard
}

func NewE07ProviderObserver(base http.RoundTripper, marker string) *E07ProviderObserver {
	budget, err := NewE07ProviderBudget(E07ProviderCallLimit)
	if err != nil {
		panic("approved E07 provider-call budget is invalid")
	}
	return NewE07ProviderObserverWithBudget(base, marker, budget)
}

func NewE07ProviderObserverWithBudget(base http.RoundTripper, marker string, budget *E07ProviderBudget) *E07ProviderObserver {
	if base == nil {
		base = http.DefaultTransport
	}
	if budget == nil {
		panic("E07 provider observer requires a shared call budget")
	}
	return &E07ProviderObserver{Base: base, Marker: marker, Budget: budget}
}

func (observer *E07ProviderObserver) RoundTrip(request *http.Request) (*http.Response, error) {
	if !observer.Budget.Reserve() {
		return nil, ErrE07ProviderBudgetExhausted
	}
	if observer.InspectReplay.Load() && observer.Marker != "" && request.Body != nil {
		payload, err := io.ReadAll(request.Body)
		_ = request.Body.Close()
		if err != nil {
			return nil, err
		}
		request.Body = io.NopCloser(bytes.NewReader(payload))

		observer.ReplayMu.Lock()
		if bytes.Contains(payload, []byte(observer.Marker)) {
			observer.Replay.PriorContextPresent = true
		}
		if bytes.Contains(payload, []byte("previous_response_id")) {
			observer.Replay.PreviousResponseIDPresent = true
		}
		observer.ReplayMu.Unlock()
	}

	observer.Budget.MarkForwarded()
	observer.Calls.Add(1)
	response, err := observer.Base.RoundTrip(request)
	observer.ObserveResponse(response, err)
	return response, err
}

func (observer *E07ProviderObserver) BudgetSnapshot() E07ProviderBudgetSnapshot {
	return observer.Budget.Snapshot()
}

func (observer *E07ProviderObserver) CloseIdleConnections() {
	if closer, ok := observer.Base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (observer *E07ProviderObserver) ArmResponseFidelityObservation() {
	observer.ResponseMu.Lock()
	observer.CaptureNext = true
	observer.Response = nil
	observer.ResponseMu.Unlock()
}

func (observer *E07ProviderObserver) TakeResponseFidelityObservation() E07ResponseFidelityObservation {
	observer.ResponseMu.Lock()
	defer observer.ResponseMu.Unlock()

	observer.CaptureNext = false
	if observer.Response == nil {
		return E07ResponseFidelityObservation{}
	}
	observation := *observer.Response
	observer.Response = nil
	return observation
}

func (observer *E07ProviderObserver) ObserveResponse(response *http.Response, responseErr error) {
	observer.ResponseMu.Lock()
	if !observer.CaptureNext {
		observer.ResponseMu.Unlock()
		return
	}
	observer.CaptureNext = false
	observation := &E07ResponseFidelityObservation{}
	if response != nil {
		observation.StatusCode = response.StatusCode
		observation.ContentType = response.Header.Get("Content-Type")
	}
	observer.Response = observation
	if responseErr != nil || response == nil || response.Body == nil {
		observer.ResponseMu.Unlock()
		return
	}
	response.Body = NewE07DigestReadCloser(response.Body, func(digest [32]byte, complete bool) {
		observer.ResponseMu.Lock()
		observation.BodyDigest = digest
		observation.BodyComplete = complete
		observer.ResponseMu.Unlock()
	})
	observer.ResponseMu.Unlock()
}

func (observer *E07ProviderObserver) Snapshot() E07ProviderObservation {
	observer.ReplayMu.Lock()
	defer observer.ReplayMu.Unlock()

	return E07ProviderObservation{Calls: observer.Calls.Load(), Replay: observer.Replay}
}
