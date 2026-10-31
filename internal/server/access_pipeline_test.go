package server_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"aihub/internal/access"
	"aihub/internal/health"
	"aihub/internal/server"
)

type pipelineProbe struct {
	called   bool
	body     []byte
	metadata access.Metadata
	found    bool
}

func (p *pipelineProbe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.called = true
	p.body, _ = io.ReadAll(r.Body)
	p.metadata, p.found = access.MetadataFrom(r.Context())
	w.WriteHeader(http.StatusOK)
}

func pipelinePolicy(t *testing.T, origins []string) access.Policy {
	t.Helper()

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey:               gatewayTestKey,
		AllowedHosts:         []string{"example.com", "localhost", "127.0.0.1", "::1"},
		AllowedHostsExplicit: true,
		AllowedOrigins:       origins,
		MaxBodyBytes:         1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy() returned error: %v", err)
	}

	return policy
}

func pipelineRouter(t *testing.T, logger *bytes.Buffer, policy access.Policy, probe http.Handler, admission access.Admission) http.Handler {
	t.Helper()

	return server.NewRouter(testLogger(logger), policy, health.Live(), health.NewRegistry().Ready(),
		server.Route{
			Method:    http.MethodPost,
			Pattern:   "/v1/responses",
			Handler:   probe,
			Admission: admission,
		},
		server.Route{
			Method:  http.MethodGet,
			Pattern: "/v1/models",
			Handler: probe,
		},
	)
}

func pipelineRequest(method, target, body string, headers map[string][]string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Host = "example.com"

	for name, values := range headers {
		request.Header[name] = values
	}

	return request
}

func authHeaders() map[string][]string {
	return map[string][]string{"Authorization": {authorization()}}
}

func TestPipelineAuthenticationPrecedesAdmission(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1024))

	request := pipelineRequest(http.MethodPost, "/v1/responses", strings.Repeat("a", 2048), map[string][]string{
		"Content-Type": {"application/json"},
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 before admission", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called for an unauthenticated request")
	}
}

func TestPipelineAuthenticatedOversizedBody(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1024))

	headers := authHeaders()
	headers["Content-Type"] = []string{"application/json"}
	request := pipelineRequest(http.MethodPost, "/v1/responses", strings.Repeat("a", 1025), headers)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called for an oversized body")
	}
}

func TestPipelineValidRequestReachesHandler(t *testing.T) {
	const body = `{"model":"deepseek-v4.1-flash"}`
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1<<20))

	headers := authHeaders()
	headers["Content-Type"] = []string{"application/json"}
	request := pipelineRequest(http.MethodPost, "/v1/responses", body, headers)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !probe.called {
		t.Fatalf("status = %d, want an authenticated handler call", recorder.Code)
	}

	if string(probe.body) != body {
		t.Fatal("accepted request bytes must be preserved")
	}

	if !probe.found || probe.metadata.Principal != access.PrincipalID {
		t.Fatalf("metadata = %+v, want an authenticated principal", probe.metadata)
	}

	pattern := regexp.MustCompile(`^aihub-ses-[0-9a-f]{32}$`)
	if !pattern.MatchString(probe.metadata.ProviderSessionID) {
		t.Fatalf("provider session = %q, want generated session metadata", probe.metadata.ProviderSessionID)
	}
}

func TestPipelineHealthAndReadiness(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1<<20))

	t.Run("get without auth", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, pipelineRequest(http.MethodGet, "/healthz", "", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", recorder.Code)
		}
	})

	t.Run("head without auth", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, pipelineRequest(http.MethodHead, "/healthz", "", nil))

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", recorder.Code)
		}
	})

	t.Run("invalid host", func(t *testing.T) {
		request := pipelineRequest(http.MethodGet, "/healthz", "", nil)
		request.Host = "evil.test"

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", recorder.Code)
		}
	})
}

func TestPipelineUnknownPath(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1<<20))

	t.Run("unauthenticated", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, pipelineRequest(http.MethodGet, "/unknown", "", nil))

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", recorder.Code)
		}
	})

	t.Run("authenticated", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, pipelineRequest(http.MethodGet, "/unknown", "", authHeaders()))

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", recorder.Code)
		}
	})
}

func TestPipelineOriginGating(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, pipelinePolicy(t, []string{"https://app.example.com"}), probe, access.ResponsesAdmission(1<<20))

	t.Run("disallowed origin", func(t *testing.T) {
		headers := authHeaders()
		headers["Origin"] = []string{"https://evil.test"}

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, pipelineRequest(http.MethodGet, "/v1/models", "", headers))

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", recorder.Code)
		}
	})

	t.Run("allowed origin", func(t *testing.T) {
		headers := authHeaders()
		headers["Origin"] = []string{"https://app.example.com"}

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, pipelineRequest(http.MethodGet, "/v1/models", "", headers))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", recorder.Code)
		}
	})
}

func TestPipelineAuthenticatedWrongMethod(t *testing.T) {
	probe := &pipelineProbe{}
	handler := pipelineRouter(t, &bytes.Buffer{}, testPolicy(t), probe, access.ResponsesAdmission(1<<20))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, pipelineRequest(http.MethodPost, "/healthz", "", authHeaders()))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 after authentication", recorder.Code)
	}
}
