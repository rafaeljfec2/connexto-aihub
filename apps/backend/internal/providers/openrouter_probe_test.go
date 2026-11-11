package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/providers"
)

func TestProbeOpenRouterEndpointUsesProtectedCredentialAndModelsRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/models" {
			t.Errorf("probe request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+openRouterTestCredential {
			t.Errorf("probe authorization header mismatch")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := providers.ProbeOpenRouterEndpoint(context.Background(), server.URL+"/api/v1", openRouterTestCredential); err != nil {
		t.Fatalf("ProbeOpenRouterEndpoint() error = %v", err)
	}
}
