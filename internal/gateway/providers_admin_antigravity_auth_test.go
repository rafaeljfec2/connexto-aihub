package gateway_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderAdminAntigravityAuthStatusAndStartProxiesToConnextoProxy(t *testing.T) {
	var statusCalls, startCalls int
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/providers/antigravity/auth/status":
			statusCalls++
			if r.Header.Get("X-Gateway-API-Key") != "fixture-gw-key" {
				t.Errorf("status: missing gateway key")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"authenticated":false,"lastCheckedAt":"2026-11-06T12:00:00Z","viabilityGate":"UNKNOWN","limitations":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/providers/antigravity/auth/start":
			startCalls++
			if r.Header.Get("X-Gateway-API-Key") != "fixture-gw-key" {
				t.Errorf("start: missing gateway key")
			}
			b, _ := io.ReadAll(r.Body)
			if string(b) != "{}" {
				t.Errorf("start body = %q, want {}", string(b))
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"operationId":"00000000-0000-4000-8000-000000000001","status":"pending","authUrl":"https://oauth.example/start","state":"state-token","expiresAt":"2026-11-06T13:00:00Z"}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(proxy.Close)

	server := adminServer(t)
	base := strings.TrimRight(proxy.URL, "/") + "/v1"
	createBody := `{"id":"antigravity","name":"Antigravity","type":"antigravity","endpoint":"` + base + `","management_credential":"fixture-gw-key"}`
	if st, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", createBody); st != http.StatusCreated {
		t.Fatalf("create antigravity status = %d", st)
	}

	st, payload := adminCall(t, http.MethodGet, server.URL+"/api/providers/antigravity/proxy/antigravity/auth/status", "")
	if st != http.StatusOK {
		t.Fatalf("auth status = %d %s", st, payload)
	}
	var status map[string]any
	if err := json.Unmarshal([]byte(payload), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["authenticated"] != false {
		t.Fatalf("authenticated = %v", status["authenticated"])
	}
	if statusCalls != 1 {
		t.Fatalf("proxy status calls = %d", statusCalls)
	}

	st, payload = adminCall(t, http.MethodPost, server.URL+"/api/providers/antigravity/proxy/antigravity/auth/start", "{}")
	if st != http.StatusOK {
		t.Fatalf("auth start = %d %s", st, payload)
	}
	var start map[string]any
	if err := json.Unmarshal([]byte(payload), &start); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	if start["auth_url"] != "https://oauth.example/start" {
		t.Fatalf("auth_url = %v", start["auth_url"])
	}
	if startCalls != 1 {
		t.Fatalf("proxy start calls = %d", startCalls)
	}
}
