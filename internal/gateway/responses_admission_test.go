package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/access"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/server"
)

const gatewayClientKey = "gateway-client-key-0123456789abcdef"

func admissionGatewayPolicy(t *testing.T) access.Policy {
	t.Helper()

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey:               gatewayClientKey,
		AllowedHosts:         []string{"127.0.0.1", "localhost", "example.com"},
		AllowedHostsExplicit: true,
		MaxBodyBytes:         1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy() returned error: %v", err)
	}

	return policy
}

func responsesThroughGatewayRouter(t *testing.T, handler http.Handler, policy access.Policy) string {
	t.Helper()

	logger, _ := newTestLogger()
	router := server.NewRouter(
		logger,
		policy,
		health.Live(),
		health.NewRegistry().Ready(),
		server.Route{
			Method:    http.MethodPost,
			Pattern:   "/v1/responses",
			Handler:   handler,
			Admission: access.ResponsesAdmission(policy.MaxBodyBytes()),
		},
	)

	return servedThroughRouter(t, router, time.Hour)
}

func authenticatedResponsesPost(t *testing.T, target, body string, headers map[string]string) (*http.Response, string) {
	t.Helper()

	merged := map[string]string{
		"Authorization": "Bearer " + gatewayClientKey,
		"Content-Type":  "application/json",
	}
	for name, value := range headers {
		merged[name] = value
	}

	return postResponses(t, target, body, merged)
}

func TestResponsesAdmissionPreservesE02Contract(t *testing.T) {
	var capture upstreamRequest

	events := []string{"event: response.completed\ndata: {}\n\n"}
	upstream := sseUpstream(t, &capture, events, 0, nil)
	logger, logBuffer := newTestLogger()
	handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, time.Minute, nil)

	policy := admissionGatewayPolicy(t)
	target := responsesThroughGatewayRouter(t, handler, policy) + "/v1/responses"

	response, body := authenticatedResponsesPost(t, target, codexStylePayload, nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	if capture.Body != codexStylePayload {
		t.Fatalf("upstream body changed in transit")
	}

	if capture.ContentType != "application/json" {
		t.Fatalf("upstream content type = %q, want application/json", capture.ContentType)
	}

	if capture.Authorization != "Bearer "+providerCredential {
		t.Fatalf("upstream authorization = %q, want the provider credential", capture.Authorization)
	}

	if !strings.HasPrefix(capture.Session, "aihub-ses-") {
		t.Fatalf("upstream session = %q, want a generated aihub session", capture.Session)
	}

	if strings.Contains(body, gatewayClientKey) || logBuffer.contains(gatewayClientKey) {
		t.Fatal("the gateway client key leaked to the response or logs")
	}
}

func TestResponsesAdmissionPassesUpstreamErrorsThrough(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "client error", status: http.StatusTooManyRequests, body: `{"error":{"type":"rate_limit"}}`},
		{name: "server error", status: http.StatusInternalServerError, body: `{"error":{"type":"server_error"}}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer upstream.Close()

			logger, _ := newTestLogger()
			handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, time.Minute, nil)

			policy := admissionGatewayPolicy(t)
			target := responsesThroughGatewayRouter(t, handler, policy) + "/v1/responses"

			response, body := authenticatedResponsesPost(t, target, codexStylePayload, nil)
			if response.StatusCode != test.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.status)
			}

			if body != test.body {
				t.Fatalf("body = %q, want the upstream envelope", body)
			}
		})
	}
}
