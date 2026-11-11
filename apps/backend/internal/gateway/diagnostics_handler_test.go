package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func TestProvidersHandlerExposesStateOnly(t *testing.T) {
	statuses := func() []providers.Status {
		return []providers.Status{{
			ID:      providers.OpenCodeGoID,
			Enabled: true,
			State:   providers.StateHealthy,
		}}
	}

	handler := gateway.NewProvidersHandler(statuses)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, newDiagnosticsRequest(t))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{"opencode-go", `"enabled":true`, `"state":"healthy"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body %q missing %q", body, want)
		}
	}

	if strings.Contains(body, "oc_sk_") || strings.Contains(body, "credential") {
		t.Fatalf("diagnostics must not expose credential material: %q", body)
	}
}

func TestProvidersHandlerReportsDisabledProvider(t *testing.T) {
	statuses := func() []providers.Status {
		return []providers.Status{{
			ID:      providers.OpenCodeGoID,
			Enabled: false,
			State:   providers.StateDisabled,
		}}
	}

	handler := gateway.NewProvidersHandler(statuses)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, newDiagnosticsRequest(t))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even for a disabled provider", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"state":"disabled"`) {
		t.Fatalf("body %q must report the disabled state", recorder.Body.String())
	}
}

func newDiagnosticsRequest(t *testing.T) *http.Request {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, "/diagnostics/providers", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	return request
}
