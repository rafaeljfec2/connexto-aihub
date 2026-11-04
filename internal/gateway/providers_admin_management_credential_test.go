package gateway_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestProviderAdminStoresManagementCredentialAsWriteOnlyConfiguration(t *testing.T) {
	server := adminServer(t)
	body := `{"id":"openrouter-primary","name":"OpenRouter Primary","type":"openrouter","endpoint":"https://provider.example.test/api/v1","credential":"inference-key-test","management_credential":"` + adminSecret + `"}`

	status, payload := adminCall(t, http.MethodPost, server.URL+"/api/providers", body)
	if status != http.StatusCreated {
		t.Fatalf("status = %d body = %s, want 201", status, payload)
	}
	if !strings.Contains(payload, `"management_credential_configured":true`) {
		t.Fatalf("response does not expose management credential presence: %s", payload)
	}
	if strings.Contains(payload, "inference-key-test") || strings.Contains(payload, "management-secret") {
		t.Fatal("provider administration response leaked a credential")
	}

	const rotatedManagementKey = "rotated-management-key"
	status, payload = adminCall(t, http.MethodPatch, server.URL+"/api/providers/openrouter-primary",
		`{"management_credential":"`+rotatedManagementKey+`"}`)
	if status != http.StatusOK || strings.Contains(payload, rotatedManagementKey) || strings.Contains(payload, "inference-key-test") {
		t.Fatalf("management credential update leaked data: status=%d body=%s", status, payload)
	}
}
