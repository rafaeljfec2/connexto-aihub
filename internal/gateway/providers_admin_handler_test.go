package gateway_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestProviderAdminCreateReturnsConfigurationWithoutCredential(t *testing.T) {
	server := adminServer(t)
	body := `{"id":"opencode-go","name":"OpenCode Go","type":"opencode-go","endpoint":"https://provider.example.test","credential":"` + adminSecret + `"}`

	status, payload := adminCall(t, http.MethodPost, server.URL+"/api/providers", body)
	if status != http.StatusCreated {
		t.Fatalf("status = %d body %s, want 201", status, payload)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if response["credential_configured"] != true || response["enabled"] != true {
		t.Fatalf("response = %v", response)
	}

	if _, leaked := response["credential"]; leaked {
		t.Fatal("response carries a credential field")
	}
}

func TestProviderAdminRejectsMalformedAndUnknownFields(t *testing.T) {
	server := adminServer(t)

	for name, body := range map[string]string{
		"malformed json": `{"id":`,
		"unknown field":  `{"id":"p1","name":"n","type":"opencode-go","endpoint":"https://e.test","surprise":1}`,
		"trailing data":  `{"id":"p1"} {"id":"p2"}`,
	} {
		if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", body); status != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", name, status)
		}
	}
}

func TestProviderAdminMapsNotFoundConflictAndState(t *testing.T) {
	server := adminServer(t)
	create := `{"id":"p1","name":"Primary","type":"opencode-go","endpoint":"https://provider.example.test"}`

	if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", create); status != http.StatusCreated {
		t.Fatalf("create status = %d", status)
	}

	if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", create); status != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", status)
	}

	if status, _ := adminCall(t, http.MethodGet, server.URL+"/api/providers/missing", ""); status != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", status)
	}

	if status, body := adminCall(t, http.MethodPost, server.URL+"/api/providers/p1/disable", ""); status != http.StatusOK || !strings.Contains(body, `"enabled":false`) {
		t.Fatalf("disable = %d %s", status, body)
	}

	if status, body := adminCall(t, http.MethodPatch, server.URL+"/api/providers/p1", `{"name":"Renamed"}`); status != http.StatusOK || !strings.Contains(body, "Renamed") {
		t.Fatalf("patch = %d %s", status, body)
	}
}

func TestProviderAdminValidateReportsMissingCredentialAndUnreachableUpstream(t *testing.T) {
	server := adminServer(t)

	if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers",
		`{"id":"p1","name":"Primary","type":"opencode-go","endpoint":"http://127.0.0.1:1"}`); status != http.StatusCreated {
		t.Fatalf("create status = %d", status)
	}

	if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers/p1/validate", ""); status != http.StatusConflict {
		t.Fatalf("validate without credential = %d, want 409", status)
	}

	if status, _ := adminCall(t, http.MethodPatch, server.URL+"/api/providers/p1",
		`{"credential":"`+adminSecret+`"}`); status != http.StatusOK {
		t.Fatalf("patch credential = %d", status)
	}

	if status, body := adminCall(t, http.MethodPost, server.URL+"/api/providers/p1/validate", ""); status != http.StatusBadGateway || !strings.Contains(body, "provider_unreachable") {
		t.Fatalf("validate unreachable = %d %s, want 502", status, body)
	}
}
