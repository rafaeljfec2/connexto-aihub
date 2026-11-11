package access_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/access"
)

const (
	gatewayFixtureKey = "gatewayKeyFixture0123456789abcdefGH"
	adminFixtureKey   = "adminKeyFixture0123456789abcdefGHIJ"
)

func authorizedStatus(t *testing.T, policy access.Policy, path, token string) int {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /api/providers", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Host = "127.0.0.1"

	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	policy.Authorize(mux).ServeHTTP(recorder, request)

	return recorder.Code
}

func TestEachKeyOnlyOpensItsOwnNamespace(t *testing.T) {
	policy, err := access.NewPolicy(access.PolicyConfig{APIKey: gatewayFixtureKey, AdminAPIKey: adminFixtureKey})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	cases := []struct {
		name  string
		path  string
		token string
		want  int
	}{
		{"admin key opens /api", "/api/providers", adminFixtureKey, http.StatusOK},
		{"gateway key refused on /api", "/api/providers", gatewayFixtureKey, http.StatusUnauthorized},
		{"gateway key opens /v1", "/v1/models", gatewayFixtureKey, http.StatusOK},
		{"admin key refused on /v1", "/v1/models", adminFixtureKey, http.StatusUnauthorized},
		{"no credential on /api", "/api/providers", "", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		if got := authorizedStatus(t, policy, tc.path, tc.token); got != tc.want {
			t.Fatalf("%s: status = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestAdministrativeNamespaceFailsClosedWithoutAdminKey(t *testing.T) {
	policy, err := access.NewPolicy(access.PolicyConfig{APIKey: gatewayFixtureKey})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	if got := authorizedStatus(t, policy, "/api/providers", gatewayFixtureKey); got != http.StatusUnauthorized {
		t.Fatalf("admin namespace without admin key: status = %d, want 401", got)
	}
}

func TestNewPolicyRejectsAdminKeyEqualToGatewayKey(t *testing.T) {
	if _, err := access.NewPolicy(access.PolicyConfig{APIKey: gatewayFixtureKey, AdminAPIKey: gatewayFixtureKey}); err == nil {
		t.Fatal("NewPolicy accepted an admin key equal to the gateway key")
	}
}
