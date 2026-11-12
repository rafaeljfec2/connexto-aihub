package userauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignupRateLimit(t *testing.T) {
	limiter := NewLimiter()
	req := httptest.NewRequest(http.MethodPost, "/api/ui/signup", nil)
	req.RemoteAddr = "198.51.100.11:12345"

	for i := 0; i < 5; i++ {
		if !limiter.AllowSignup(req) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if limiter.AllowSignup(req) {
		t.Fatal("sixth attempt should be blocked")
	}
}

func TestLoginRateLimitIndependentFromSignup(t *testing.T) {
	limiter := NewLimiter()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/ui/login", nil)
	loginReq.RemoteAddr = "198.51.100.12:12345"
	signupReq := httptest.NewRequest(http.MethodPost, "/api/ui/signup", nil)
	signupReq.RemoteAddr = "198.51.100.12:12345"

	for i := 0; i < 5; i++ {
		if !limiter.AllowLogin(loginReq) {
			t.Fatalf("login attempt %d should be allowed", i+1)
		}
	}
	if !limiter.AllowSignup(signupReq) {
		t.Fatal("signup should not consume login window")
	}
}

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
