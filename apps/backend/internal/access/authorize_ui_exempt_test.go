package access

import (
	"net/http/httptest"
	"testing"
)

func TestUIAuthExemptRoutes(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{"GET", "/api/ui/session", true},
		{"POST", "/api/ui/login", true},
		{"POST", "/api/ui/signup", true},
		{"POST", "/api/ui/refresh", true},
		{"POST", "/api/ui/forgot-password", true},
		{"POST", "/api/ui/reset-password", true},
		{"GET", "/api/ui/login", false},
		{"POST", "/api/ui/logout", false},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if got := uiAuthExempt(req); got != tc.want {
			t.Fatalf("%s %s exempt=%v want %v", tc.method, tc.path, got, tc.want)
		}
	}
}
