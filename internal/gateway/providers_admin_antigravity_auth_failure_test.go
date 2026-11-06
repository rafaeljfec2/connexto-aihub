package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderAdminAntigravityAuthStatusSurfacesUpstreamDetail(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers/antigravity/auth/status" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"type":"gateway_auth","message":"invalid gateway key"}}`))
	}))
	t.Cleanup(proxy.Close)

	server := adminServer(t)
	base := strings.TrimRight(proxy.URL, "/") + "/v1"
	createBody := `{"id":"antigravity","name":"Antigravity","type":"antigravity","endpoint":"` + base + `","management_credential":"gw-key"}`
	if st, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", createBody); st != http.StatusCreated {
		t.Fatalf("create status = %d", st)
	}

	st, payload := adminCall(t, http.MethodGet, server.URL+"/api/providers/antigravity/proxy/antigravity/auth/status", "")
	if st != http.StatusBadGateway {
		t.Fatalf("status = %d body %s", st, payload)
	}
	if !strings.Contains(payload, "invalid gateway key") {
		t.Fatalf("expected upstream detail in body: %s", payload)
	}
}
