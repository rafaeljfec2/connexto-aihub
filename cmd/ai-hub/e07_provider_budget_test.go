package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const e07ExpectedProviderCallLimit = e07ProviderCallLimit

func TestE07ProviderBudgetRejectsCallsBeforeUnderlyingTransport(t *testing.T) {
	var received atomic.Int64
	base := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		received.Add(1)
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header),
			Body: http.NoBody, Request: request}, nil
	})
	observer := newE07ProviderObserver(base, "")

	for attempt := range e07ExpectedProviderCallLimit + 1 {
		request := httptest.NewRequest(http.MethodGet, "http://provider.invalid/v1/models", nil)
		_, err := observer.RoundTrip(request)
		if attempt < e07ExpectedProviderCallLimit && err != nil {
			t.Fatal("an in-budget provider call was rejected")
		}
		if attempt == e07ExpectedProviderCallLimit && !errors.Is(err, errE07ProviderBudgetExhausted) {
			t.Fatal("an over-budget provider call was not rejected locally")
		}
	}

	if got := received.Load(); got != e07ExpectedProviderCallLimit {
		t.Fatalf("underlying transport received %d calls, want exactly %d", got, e07ExpectedProviderCallLimit)
	}
	assertE07BudgetSnapshot(t, observer.BudgetSnapshot(), 10, 9, 1, 9)
}

func TestE07ProviderBudgetNeverExceedsLimitUnderConcurrency(t *testing.T) {
	var received atomic.Int64
	base := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		received.Add(1)
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header),
			Body: http.NoBody, Request: request}, nil
	})
	observer := newE07ProviderObserver(base, "")
	const attempts = 50

	var wait sync.WaitGroup
	for range attempts {
		wait.Add(1)
		go func() {
			defer wait.Done()
			request := httptest.NewRequest(http.MethodGet, "http://provider.invalid/v1/models", nil)
			_, _ = observer.RoundTrip(request)
		}()
	}
	wait.Wait()

	if got := received.Load(); got != e07ExpectedProviderCallLimit {
		t.Fatalf("underlying transport received %d concurrent calls, want %d", got, e07ExpectedProviderCallLimit)
	}
	assertE07BudgetSnapshot(t, observer.BudgetSnapshot(), attempts, 9, attempts-9, 9)
}

func TestE07ProviderBudgetCountsRedirectRetryAgainstSameBudget(t *testing.T) {
	var received atomic.Int64
	base := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		received.Add(1)
		return &http.Response{StatusCode: http.StatusTemporaryRedirect,
			Header: http.Header{"Location": []string{"http://provider.invalid/retry"}},
			Body:   http.NoBody, Request: request}, nil
	})
	observer := newE07ProviderObserver(base, "")
	request, err := http.NewRequest(http.MethodGet, "http://provider.invalid/start", nil)
	if err != nil {
		t.Fatal("client request could not be created")
	}
	_, requestErr := (&http.Client{Transport: observer}).Do(request)
	if requestErr == nil {
		t.Fatal("redirect retry passed after the provider-call budget was exhausted")
	}
	if got := received.Load(); got != e07ExpectedProviderCallLimit {
		t.Fatalf("underlying transport received %d calls after redirects, want %d", got, e07ExpectedProviderCallLimit)
	}
	assertE07BudgetSnapshot(t, observer.BudgetSnapshot(), 10, 9, 1, 9)
}

func TestE07ProviderBudgetIncludesHubReadinessAndExcludesUnauthenticatedRequest(t *testing.T) {
	hub, observer, cfg, err := e07StartAuthProbeHub(t)
	if err != nil {
		t.Fatal("local validation Hub could not start")
	}
	hubStopped := false
	t.Cleanup(func() {
		if !hubStopped {
			if err := hub.stop(); err != nil {
				t.Error("local validation Hub did not stop")
			}
		}
	})

	startup := observer.BudgetSnapshot()
	if startup.Forwarded != 1 || startup.Attempted != 1 {
		t.Fatalf("provider readiness did not consume the shared budget: %+v", startup)
	}
	status, unchanged, _ := e07UnauthenticatedProbe(hub, observer)
	if status != http.StatusUnauthorized || !unchanged {
		t.Fatalf("unauthenticated probe status=%d provider_unchanged=%t", status, unchanged)
	}
	if got := observer.BudgetSnapshot(); got != startup {
		t.Fatalf("unauthenticated request changed provider budget: before=%+v after=%+v", startup, got)
	}
	if err := hub.stop(); err != nil {
		t.Fatal("local validation Hub did not stop before restart")
	}
	hubStopped = true
	restarted, err := startE07Hub(cfg, observer, e07DiscardLogger())
	if err != nil {
		t.Fatal("local validation Hub could not restart with the shared budget")
	}
	hub = restarted
	hubStopped = false
	if got := observer.BudgetSnapshot(); got.Forwarded != 2 || got.Attempted != 2 {
		t.Fatalf("provider readiness budget was reset across restart: %+v", got)
	}
}

func TestE07ProviderBudgetIsSharedAcrossObserverInstances(t *testing.T) {
	var received atomic.Int64
	base := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		received.Add(1)
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header),
			Body: http.NoBody, Request: request}, nil
	})
	budget, err := newE07ProviderBudget(2)
	if err != nil {
		t.Fatal("valid provider-call budget was rejected")
	}
	firstScenario := newE07ProviderObserverWithBudget(base, "", budget)
	secondScenario := newE07ProviderObserverWithBudget(base, "", budget)

	for _, observer := range []*e07ProviderObserver{firstScenario, secondScenario, secondScenario} {
		request := httptest.NewRequest(http.MethodGet, "http://provider.invalid/v1/models", nil)
		_, _ = observer.RoundTrip(request)
	}

	if got := received.Load(); got != 2 {
		t.Fatalf("shared transport received %d calls across scenarios, want 2", got)
	}
	assertE07BudgetSnapshot(t, budget.Snapshot(), 3, 2, 1, 2)
}

func TestE07ProviderBudgetRejectsLimitsOutsideApprovedPlan(t *testing.T) {
	for _, limit := range []int{0, e07ProviderCallLimit + 1} {
		if _, err := newE07ProviderBudget(limit); err == nil {
			t.Fatalf("provider-call budget limit %d was accepted", limit)
		}
	}
}

func TestE07ProviderTransportDoesNotRetryAcrossConnections(t *testing.T) {
	transport, err := newE07ProviderTransport()
	if err != nil {
		t.Fatal("validation provider transport could not be created")
	}
	defer transport.CloseIdleConnections()
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		hijacker, ok := writer.(http.Hijacker)
		if !ok {
			t.Error("HTTP test server does not support connection hijacking")
			return
		}
		connection, _, err := hijacker.Hijack()
		if err != nil {
			t.Error("HTTP test server could not close the request connection")
			return
		}
		_ = connection.Close()
	}))
	defer server.Close()
	client := &http.Client{Transport: transport}
	firstResponse, err := client.Get(server.URL)
	if err != nil {
		t.Fatal("initial provider request failed")
	}
	_ = firstResponse.Body.Close()
	if _, err := client.Get(server.URL); err == nil {
		t.Fatal("provider transport retried a failed request automatically")
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("HTTP server received %d requests, want 2 without a retry", got)
	}
}

func assertE07BudgetSnapshot(t *testing.T, got e07ProviderBudgetSnapshot, attempted, admitted, rejected, forwarded int) {
	t.Helper()
	if got.Limit < 1 || got.Limit > e07ProviderCallLimit || got.Attempted != attempted ||
		got.Admitted != admitted || got.Rejected != rejected || got.Forwarded != forwarded {
		t.Fatalf("provider-call budget snapshot = %+v", got)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (roundTripper roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTripper(request)
}

var _ http.RoundTripper = roundTripperFunc(nil)

func TestE07ProviderHTTPClientCancelsRequestAtTimeout(t *testing.T) {
	var canceled atomic.Bool
	base := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		canceled.Store(true)
		return nil, request.Context().Err()
	})
	observer := newE07ProviderObserver(base, "")
	client := e07ProviderHTTPClient(observer, 20*time.Millisecond)
	request, err := http.NewRequest(http.MethodGet, "http://provider.invalid/v1/models", nil)
	if err != nil {
		t.Fatal("validation request could not be created")
	}
	response, err := client.Do(request)
	if response != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("provider HTTP client did not enforce its request timeout")
	}
	if !canceled.Load() {
		t.Fatal("provider transport did not observe timeout cancellation")
	}
	if got := observer.BudgetSnapshot().Forwarded; got != 1 {
		t.Fatalf("timed provider request forwarded %d calls, want 1", got)
	}
}
