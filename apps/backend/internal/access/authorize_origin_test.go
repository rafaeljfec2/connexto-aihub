package access_test

import (
	"net/http"
	"testing"

	"aihub/internal/access"
)

func TestAuthorizeOrigin(t *testing.T) {
	tests := []struct {
		name       string
		origins    []string
		wantStatus int
	}{
		{name: "absent", wantStatus: http.StatusOK},
		{name: "exact", origins: []string{allowedOrigin}, wantStatus: http.StatusOK},
		{name: "case and default port", origins: []string{"HTTPS://APP.EXAMPLE.COM:443"}, wantStatus: http.StatusOK},
		{name: "disallowed", origins: []string{"https://evil.test"}, wantStatus: http.StatusForbidden},
		{name: "null", origins: []string{"null"}, wantStatus: http.StatusForbidden},
		{name: "wildcard", origins: []string{"*"}, wantStatus: http.StatusForbidden},
		{name: "path", origins: []string{allowedOrigin + "/path"}, wantStatus: http.StatusForbidden},
		{name: "empty port", origins: []string{allowedOrigin + ":"}, wantStatus: http.StatusForbidden},
		{name: "empty query", origins: []string{allowedOrigin + "?"}, wantStatus: http.StatusForbidden},
		{name: "empty fragment", origins: []string{allowedOrigin + "#"}, wantStatus: http.StatusForbidden},
		{name: "two values", origins: []string{allowedOrigin, allowedOrigin}, wantStatus: http.StatusForbidden},
	}

	policy := testPolicy(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers := validAuth()
			if test.origins != nil {
				headers["Origin"] = test.origins
			}

			recorder, _ := serve(t, policy, testRequest(http.MethodGet, "/v1/models", "localhost", headers))
			if got := recorder.Code; got != test.wantStatus {
				t.Fatalf("status = %d, want %d", got, test.wantStatus)
			}
		})
	}
}

func TestAuthorizeSetsMetadata(t *testing.T) {
	policy := testPolicy(t)
	headers := validAuth()
	headers["Origin"] = []string{allowedOrigin}

	_, probe := serve(t, policy, testRequest(http.MethodGet, "/v1/models", "LOCALHOST.:8080", headers))
	if !probe.found {
		t.Fatal("metadata envelope must be present")
	}

	if probe.metadata.Principal != access.PrincipalID {
		t.Fatalf("principal = %q, want %q", probe.metadata.Principal, access.PrincipalID)
	}

	if probe.metadata.Host != "localhost" {
		t.Fatalf("host = %q, want localhost", probe.metadata.Host)
	}

	if probe.metadata.Origin != allowedOrigin {
		t.Fatalf("origin = %q, want %q", probe.metadata.Origin, allowedOrigin)
	}
}

func TestAuthorizeHealthMetadataHasNoPrincipal(t *testing.T) {
	policy := testPolicy(t)

	_, probe := serve(t, policy, testRequest(http.MethodGet, "/healthz", "localhost", nil))
	if !probe.found {
		t.Fatal("metadata envelope must be present")
	}

	if probe.metadata.Principal != "" {
		t.Fatalf("principal = %q, want empty for unauthenticated health", probe.metadata.Principal)
	}

	if probe.metadata.Host != "localhost" {
		t.Fatalf("host = %q, want localhost", probe.metadata.Host)
	}
}
