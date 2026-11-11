package access_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/access"
)

const allowedOrigin = "https://app.example.com"

func testPolicy(t *testing.T) access.Policy {
	t.Helper()

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey:               testGatewayKey,
		AllowedHosts:         []string{"localhost", "127.0.0.1", "::1", "example.com"},
		AllowedHostsExplicit: true,
		AllowedOrigins:       []string{allowedOrigin},
		MaxBodyBytes:         1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy() returned error: %v", err)
	}

	return policy
}

type probeHandler struct {
	called   bool
	metadata access.Metadata
	found    bool
}

func (p *probeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.called = true
	p.metadata, p.found = access.MetadataFrom(r.Context())
	w.WriteHeader(http.StatusOK)
}

func testRequest(method, target, host string, headers map[string][]string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	request.Host = host

	for name, values := range headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}

	return request
}

func validAuth() map[string][]string {
	return map[string][]string{"Authorization": {"Bearer " + testGatewayKey}}
}

func serve(t *testing.T, policy access.Policy, request *http.Request) (*httptest.ResponseRecorder, *probeHandler) {
	t.Helper()

	probe := &probeHandler{}
	recorder := httptest.NewRecorder()
	policy.Authorize(probe).ServeHTTP(recorder, request)

	return recorder, probe
}

func TestAuthorizePrecedence(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		headers    map[string][]string
		wantStatus int
		wantCalled bool
	}{
		{name: "invalid host beats missing key", host: "evil.test", wantStatus: http.StatusForbidden},
		{name: "invalid host beats valid key", host: "evil.test", headers: validAuth(), wantStatus: http.StatusForbidden},
		{name: "valid host and missing key", host: "localhost", wantStatus: http.StatusUnauthorized},
		{name: "valid host and valid key", host: "localhost", headers: validAuth(), wantStatus: http.StatusOK, wantCalled: true},
	}

	policy := testPolicy(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder, probe := serve(t, policy, testRequest(http.MethodGet, "/v1/models", test.host, test.headers))

			if got := recorder.Code; got != test.wantStatus {
				t.Fatalf("status = %d, want %d", got, test.wantStatus)
			}

			if probe.called != test.wantCalled {
				t.Fatalf("handler called = %v, want %v", probe.called, test.wantCalled)
			}
		})
	}
}

func TestAuthorizeHealthExemption(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "liveness get", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusOK},
		{name: "readiness get", method: http.MethodGet, path: "/readyz", wantStatus: http.StatusOK},
		{name: "liveness head", method: http.MethodHead, path: "/healthz", wantStatus: http.StatusUnauthorized},
		{name: "readiness head", method: http.MethodHead, path: "/readyz", wantStatus: http.StatusUnauthorized},
		{name: "liveness post", method: http.MethodPost, path: "/healthz", wantStatus: http.StatusUnauthorized},
	}

	policy := testPolicy(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder, _ := serve(t, policy, testRequest(test.method, test.path, "localhost", nil))
			if got := recorder.Code; got != test.wantStatus {
				t.Fatalf("status = %d, want %d", got, test.wantStatus)
			}
		})
	}
}

func TestAuthorizeNormalizesHost(t *testing.T) {
	policy := testPolicy(t)

	for _, host := range []string{"LOCALHOST.:8080", "127.0.0.1:9999", "[::1]:8080", "example.com"} {
		t.Run(host, func(t *testing.T) {
			recorder, _ := serve(t, policy, testRequest(http.MethodGet, "/v1/models", host, validAuth()))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 for host %q", recorder.Code, host)
			}
		})
	}
}

func TestAuthorizeRejectsMalformedHosts(t *testing.T) {
	policy := testPolicy(t)

	for _, host := range []string{"", "evil.test", "user@localhost", "localhost/path", "*.localhost", "localhost:bad", "localhost:+80", "localhost:-0", "[::1"} {
		t.Run(host, func(t *testing.T) {
			recorder, probe := serve(t, policy, testRequest(http.MethodGet, "/v1/models", host, validAuth()))
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 for host %q", recorder.Code, host)
			}
			if probe.called {
				t.Fatal("handler must not be called for a rejected host")
			}
		})
	}
}

func TestAuthorizeBearerForms(t *testing.T) {
	tests := []struct {
		name       string
		values     []string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "basic", values: []string{"Basic " + testGatewayKey}, wantStatus: http.StatusUnauthorized},
		{name: "empty token", values: []string{"Bearer "}, wantStatus: http.StatusUnauthorized},
		{name: "two spaces", values: []string{"Bearer  " + testGatewayKey}, wantStatus: http.StatusUnauthorized},
		{name: "tab", values: []string{"Bearer\t" + testGatewayKey}, wantStatus: http.StatusUnauthorized},
		{name: "comma list", values: []string{"Bearer " + testGatewayKey + ",other"}, wantStatus: http.StatusUnauthorized},
		{name: "wrong key", values: []string{"Bearer wrong-key-0123456789abcdefghij"}, wantStatus: http.StatusUnauthorized},
		{name: "duplicate", values: []string{"Bearer " + testGatewayKey, "Bearer " + testGatewayKey}, wantStatus: http.StatusUnauthorized},
		{name: "lowercase scheme", values: []string{"bearer " + testGatewayKey}, wantStatus: http.StatusOK},
		{name: "uppercase scheme", values: []string{"BEARER " + testGatewayKey}, wantStatus: http.StatusOK},
	}

	policy := testPolicy(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers := map[string][]string{}
			if test.values != nil {
				headers["Authorization"] = test.values
			}

			recorder, _ := serve(t, policy, testRequest(http.MethodGet, "/v1/models", "localhost", headers))
			if got := recorder.Code; got != test.wantStatus {
				t.Fatalf("status = %d, want %d", got, test.wantStatus)
			}
		})
	}
}

func assertEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, wantType string) {
	t.Helper()

	if got, want := recorder.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}

	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}

	if envelope.Error.Type != wantType {
		t.Fatalf("error type = %q, want %q", envelope.Error.Type, wantType)
	}

	if envelope.Error.Message == "" {
		t.Fatal("error message must not be empty")
	}
}

func TestAuthorizeErrorContract(t *testing.T) {
	policy := testPolicy(t)

	t.Run("unauthorized", func(t *testing.T) {
		recorder, _ := serve(t, policy, testRequest(http.MethodGet, "/v1/models", "localhost", nil))

		if got, want := recorder.Header().Get("WWW-Authenticate"), `Bearer realm="ai-hub"`; got != want {
			t.Fatalf("WWW-Authenticate = %q, want %q", got, want)
		}

		if got, want := recorder.Header().Get("Cache-Control"), "no-store"; got != want {
			t.Fatalf("Cache-Control = %q, want %q", got, want)
		}

		assertEnvelope(t, recorder, "unauthorized")
		if strings.Contains(recorder.Body.String(), testGatewayKey) {
			t.Fatal("response must not contain the gateway key")
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		recorder, _ := serve(t, policy, testRequest(http.MethodGet, "/v1/models", "evil.test", validAuth()))

		if got := recorder.Header().Get("WWW-Authenticate"); got != "" {
			t.Fatalf("WWW-Authenticate = %q, want empty", got)
		}

		assertEnvelope(t, recorder, "forbidden")
		if strings.Contains(recorder.Body.String(), "evil.test") {
			t.Fatal("response must not echo the rejected host")
		}
	})
}
