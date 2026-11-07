package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderAdminCursorAuthStartSurfacesBrowserUnavailable(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers/cursor/auth" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"browser_unavailable","message":"Chromium is unavailable"}`))
	}))
	t.Cleanup(proxy.Close)

	server := adminServer(t)
	base := strings.TrimRight(proxy.URL, "/") + "/v1"
	createBody := `{"id":"cursor","name":"Cursor","type":"cursor","endpoint":"` + base + `","management_credential":"gw-key"}`
	if st, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", createBody); st != http.StatusCreated {
		t.Fatalf("create status = %d", st)
	}

	st, payload := adminCall(t, http.MethodPost, server.URL+"/api/providers/cursor/proxy/cursor/auth/start", "")
	if st != http.StatusBadGateway {
		t.Fatalf("status = %d body %s", st, payload)
	}
	if !strings.Contains(payload, "proxy_cursor_auth_failed") {
		t.Fatalf("expected proxy_cursor_auth_failed type in body: %s", payload)
	}
	if strings.Contains(strings.ToLower(payload), "antigravity") {
		t.Fatalf("must not mention Antigravity: %s", payload)
	}
	if !strings.Contains(payload, "CHROMIUM_EXECUTABLE_PATH") {
		t.Fatalf("expected Chromium hint in body: %s", payload)
	}
}
