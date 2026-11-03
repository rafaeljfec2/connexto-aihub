package gateway_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/access"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/operational"
	"aihub/internal/providers"
)

func TestOperationalEventsRequiresAdminAuth(t *testing.T) {
	hub := operational.NewHub(nil)
	readiness := health.NewRegistry()
	handler := gateway.NewOperationalEventsHandler(hub, readiness, func() []providers.Status { return nil }, nil)

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey:               "gateway-key-123456789012345678901234",
		AdminAPIKey:          "admin-key-123456789012345678901234",
		AllowedHosts:         []string{"127.0.0.1"},
		AllowedHostsExplicit: true,
	})
	if err != nil {
		t.Fatalf("policy: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/events", handler)
	protected := policy.Authorize(mux)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Host = "127.0.0.1"
	protected.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestOperationalEventsStreamsInitialSnapshot(t *testing.T) {
	hub := operational.NewHub(nil)
	readiness := health.NewRegistry()
	statuses := func() []providers.Status {
		return []providers.Status{{
			ID:      providers.OpenCodeGoID,
			Enabled: true,
			State:   providers.StateHealthy,
		}}
	}
	handler := gateway.NewOperationalEventsHandler(hub, readiness, statuses, nil)

	server := httptest.NewServer(handler)
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer response.Body.Close()

	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("Content-Type = %q", response.Header.Get("Content-Type"))
	}

	buf := make([]byte, 512)
	var text strings.Builder
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(text.String(), "event: provider_status") {
		n, readErr := response.Body.Read(buf)
		if n > 0 {
			text.Write(buf[:n])
		}
		if readErr != nil && readErr != io.EOF {
			t.Fatalf("read body: %v", readErr)
		}
		if n == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	bodyText := text.String()
	for _, want := range []string{"event: readiness_changed", "event: provider_status", `"schema_version":1`, providers.OpenCodeGoID} {
		if !strings.Contains(bodyText, want) {
			t.Fatalf("body %q missing %q", bodyText, want)
		}
	}
}
