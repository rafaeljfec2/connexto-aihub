package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/ui"
)

func TestUILoginHandlerSetsCookie(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = 'x'
	}

	gate, err := ui.NewGate(ui.Settings{
		Passphrase:    "test-operator-passphrase-min-length",
		SessionSecret: secret,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := NewUILoginHandler(gate, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ui/login", strings.NewReader(`{"passphrase":"test-operator-passphrase-min-length"}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, ui.CookieName) {
		t.Fatalf("set-cookie = %q", setCookie)
	}
}
