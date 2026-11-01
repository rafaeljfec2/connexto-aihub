package main

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

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
