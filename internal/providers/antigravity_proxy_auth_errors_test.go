package providers

import (
	"errors"
	"testing"
)

func TestProxyAntigravityAuthFailedErrorIsSentinel(t *testing.T) {
	err := newProxyAntigravityAuthFailed(502, []byte(`{"error":{"type":"upstream","message":"rate limited"}}`))
	if !errors.Is(err, ErrProxyAntigravityAuthFailed) {
		t.Fatalf("errors.Is = false, want true for sentinel")
	}
	var authErr *ProxyAntigravityAuthFailedError
	if !errors.As(err, &authErr) {
		t.Fatalf("errors.As to ProxyAntigravityAuthFailedError = false")
	}
	if authErr.Status != 502 {
		t.Fatalf("status = %d", authErr.Status)
	}
	if authErr.Detail == "" || authErr.Detail != "rate limited" {
		t.Fatalf("detail = %q", authErr.Detail)
	}
}

func TestExtractProxyAntigravityAuthUpstreamDetail_plainJSONMessage(t *testing.T) {
	got := extractProxyAntigravityAuthUpstreamDetail([]byte(`{"message":"bad token"}`))
	if got != "bad token" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractProxyAntigravityAuthUpstreamDetail_sanitizesControls(t *testing.T) {
	got := extractProxyAntigravityAuthUpstreamDetail([]byte("line1\nline2\x00x"))
	if got != "line1 line2x" {
		t.Fatalf("got %q", got)
	}
}
