package support

import (
	"net/http"
	"testing"
)

const E07ExpectedProviderCallLimit = E07ProviderCallLimit

func AssertE07BudgetSnapshot(t *testing.T, got E07ProviderBudgetSnapshot, attempted, admitted, rejected, forwarded int) {
	t.Helper()
	if got.Limit < 1 || got.Limit > E07ProviderCallLimit || got.Attempted != attempted ||
		got.Admitted != admitted || got.Rejected != rejected || got.Forwarded != forwarded {
		t.Fatalf("provider-call budget snapshot = %+v", got)
	}
}

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (roundTripper RoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTripper(request)
}

var _ http.RoundTripper = RoundTripperFunc(nil)
