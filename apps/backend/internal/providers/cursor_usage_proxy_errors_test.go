package providers

import (
	"errors"
	"testing"
)

func TestProxyCursorUsageFailedError_browser_unavailable(t *testing.T) {
	body := []byte(`{"status":"browser_unavailable","message":"Chromium is unavailable"}`)
	err := newProxyCursorUsageFailed(503, body)
	if !errors.Is(err, ErrProxyCursorUsageFailed) {
		t.Fatalf("errors.Is = false, want true for sentinel")
	}
	var usageErr *ProxyCursorUsageFailedError
	if !errors.As(err, &usageErr) {
		t.Fatalf("errors.As to ProxyCursorUsageFailedError = false")
	}
	if usageErr.Status != 503 {
		t.Fatalf("status = %d", usageErr.Status)
	}
	if usageErr.Code != "browser_unavailable" {
		t.Fatalf("code = %q", usageErr.Code)
	}
	if usageErr.Detail != "Chromium is unavailable" {
		t.Fatalf("detail = %q", usageErr.Detail)
	}
}
