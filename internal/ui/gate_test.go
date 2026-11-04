package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoopbackTrustWithoutPassphrase(t *testing.T) {
	gate, err := NewGate(Settings{LoopbackServer: true})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	if !gate.Authenticated(request) {
		t.Fatal("expected loopback trust")
	}

	info := gate.Session(request)
	if info.Mode != "loopback_trust" || !info.Authenticated {
		t.Fatalf("session = %+v", info)
	}
}

func TestPassphraseLoginSessionCookie(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte('a' + (i % 26))
	}

	gate, err := NewGate(Settings{
		Passphrase:     "operator-secret-passphrase-value",
		SessionSecret:  secret,
		LoopbackServer: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	setCookie, err := gate.Login("operator-secret-passphrase-value")
	if err != nil {
		t.Fatal(err)
	}

	token := cookieValueFromSetCookie(setCookie)
	request := httptest.NewRequest(http.MethodGet, "/api/resource-policies", nil)
	request.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	if !gate.Authenticated(request) {
		t.Fatal("expected authenticated session")
	}
}

func TestRemoteClientDeniedWithoutPassphrase(t *testing.T) {
	gate, err := NewGate(Settings{LoopbackServer: true})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/resource-policies", nil)
	request.RemoteAddr = "203.0.113.50:54321"
	if gate.Authenticated(request) {
		t.Fatal("expected remote client denied without passphrase session")
	}

	info := gate.Session(request)
	if info.Authenticated || info.Mode != "none" {
		t.Fatalf("session = %+v", info)
	}
}

func TestPassphraseDisablesLoopbackTrust(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte('b')
	}

	gate, err := NewGate(Settings{
		Passphrase:     "test-passphrase-minimum-length-value",
		SessionSecret:  secret,
		LoopbackServer: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if gate.LoopbackTrustEnabled() {
		t.Fatal("loopback trust must be disabled when passphrase is configured")
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	if gate.Authenticated(request) {
		t.Fatal("loopback TCP must not bypass passphrase mode")
	}
}

func cookieValueFromSetCookie(setCookie string) string {
	const prefix = CookieName + "="
	start := strings.Index(setCookie, prefix)
	if start < 0 {
		return ""
	}
	value := setCookie[start+len(prefix):]
	if semi := strings.Index(value, ";"); semi >= 0 {
		value = value[:semi]
	}
	return value
}
