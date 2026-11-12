package userauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginRateLimit(t *testing.T) {
	limiter := NewLimiter()
	req := httptest.NewRequest(http.MethodPost, "/api/ui/login", nil)
	req.RemoteAddr = "198.51.100.10:12345"

	for i := 0; i < 5; i++ {
		if !limiter.AllowLogin(req) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if limiter.AllowLogin(req) {
		t.Fatal("sixth attempt should be blocked")
	}
}
