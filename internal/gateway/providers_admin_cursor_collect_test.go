package gateway_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCursorProxyUsageCollectForwardsCollectStatus(t *testing.T) {
	t.Parallel()

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "usage/collect"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":    "authentication_required",
				"provider":  "cursor",
				"accountId": "cursor",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(proxy.Close)

	server := adminServer(t)
	base := strings.TrimRight(proxy.URL, "/") + "/v1"
	createBody := `{"id":"cursor","name":"Cursor","type":"cursor","endpoint":"` + base + `","management_credential":"gw-key"}`
	if st, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", createBody); st != http.StatusCreated {
		t.Fatalf("create status = %d", st)
	}

	st, body := adminCall(t, http.MethodPost, server.URL+"/api/providers/cursor/proxy/cursor/usage/collect", "")
	if st != http.StatusOK {
		t.Fatalf("collect status = %d body %s", st, body)
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != "authentication_required" {
		t.Fatalf("status = %q, want authentication_required", payload.Status)
	}
}
