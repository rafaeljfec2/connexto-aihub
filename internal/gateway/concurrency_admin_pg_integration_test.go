package gateway_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"aihub/internal/access"
	"aihub/internal/concurrency"
	"aihub/internal/concurrency/pgstore"
	"aihub/internal/gateway"
)

func concurrencyAdminPGServer(t *testing.T) *httptest.Server {
	t.Helper()
	pool := e2eMigratedPool(t)
	service := concurrency.NewService(pgstore.New(pool))
	admin := gateway.NewConcurrencyAdmin(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey: resourceGatewayToken, AdminAPIKey: resourceAdminToken, MaxBodyBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/concurrency-policies", admin.List())
	mux.Handle("PUT /api/concurrency-policies", admin.Upsert())
	mux.Handle("GET /api/concurrency-usage", admin.Usage())
	mux.Handle("POST /api/concurrency-policies/{id}/enable", admin.Enable())
	mux.Handle("POST /api/concurrency-policies/{id}/disable", admin.Disable())
	server := httptest.NewServer(policy.Authorize(mux))
	t.Cleanup(server.Close)
	return server
}

func TestE0931ScenarioFConcurrencyAdminHTTPIntegration(t *testing.T) {
	server := concurrencyAdminPGServer(t)
	token := resourceAdminToken

	if status, _ := resourceAdminCall(t, http.MethodGet, server.URL+"/api/concurrency-policies", "", ""); status != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d", status)
	}
	if status, _ := resourceAdminCall(t, http.MethodGet, server.URL+"/api/concurrency-policies", "", "not-a-valid-token"); status != http.StatusUnauthorized {
		t.Fatalf("invalid token status = %d", status)
	}

	createBody := `{"scope":"global","concurrency_limit":4,"enabled":true}`
	status, payload := resourceAdminCall(t, http.MethodPut, server.URL+"/api/concurrency-policies", createBody, token)
	if status != http.StatusCreated || !strings.Contains(payload, `"concurrency_limit":4`) {
		t.Fatalf("create = %d %s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodPut, server.URL+"/api/concurrency-policies", createBody, token)
	if status != http.StatusOK {
		t.Fatalf("update = %d %s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodGet, server.URL+"/api/concurrency-policies", "", token)
	if status != http.StatusOK || !strings.Contains(payload, `"policies"`) {
		t.Fatalf("list = %d %s", status, payload)
	}

	var listed struct {
		Policies []struct {
			ID int64 `json:"id"`
		} `json:"policies"`
	}
	if err := json.Unmarshal([]byte(payload), &listed); err != nil || len(listed.Policies) != 1 {
		t.Fatalf("list decode: %v policies=%+v", err, listed.Policies)
	}
	id := listed.Policies[0].ID

	disableURL := server.URL + "/api/concurrency-policies/" + strconv.FormatInt(id, 10) + "/disable"
	status, payload = resourceAdminCall(t, http.MethodPost, disableURL, "", token)
	if status != http.StatusOK || !strings.Contains(payload, `"enabled":false`) {
		t.Fatalf("disable = %d %s", status, payload)
	}
	enableURL := server.URL + "/api/concurrency-policies/" + strconv.FormatInt(id, 10) + "/enable"
	status, payload = resourceAdminCall(t, http.MethodPost, enableURL, "", token)
	if status != http.StatusOK || !strings.Contains(payload, `"enabled":true`) {
		t.Fatalf("enable = %d %s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodGet, server.URL+"/api/concurrency-usage", "", token)
	if status != http.StatusOK || !strings.Contains(payload, `"active_count"`) {
		t.Fatalf("usage = %d %s", status, payload)
	}

	for name, body := range map[string]string{
		"invalid limit":  `{"scope":"global","concurrency_limit":0}`,
		"invalid scope":  `{"scope":"tenant","concurrency_limit":1}`,
		"unknown field":  `{"scope":"global","concurrency_limit":1,"extra":1}`,
		"provider scope": `{"scope":"provider","concurrency_limit":1}`,
	} {
		status, _ := resourceAdminCall(t, http.MethodPut, server.URL+"/api/concurrency-policies", body, token)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: status = %d", name, status)
		}
	}

	if status, _ := resourceAdminCall(t, http.MethodGet, server.URL+"/api/concurrency-usage?unknown=1", "", token); status != http.StatusBadRequest {
		t.Fatalf("unknown query status = %d", status)
	}

	largeBody := strings.Repeat("x", 5000)
	status, _ = resourceAdminCall(t, http.MethodPut, server.URL+"/api/concurrency-policies", `{"scope":"global","concurrency_limit":1,"enabled":true,"padding":"`+largeBody+`"}`, token)
	if status != http.StatusBadRequest {
		t.Fatalf("oversized body status = %d", status)
	}
}
