package access

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/ui"
)

type gateStub struct {
	ok bool
}

func (g gateStub) Authenticated(*http.Request) bool { return g.ok }

func TestAuthorizeAcceptsLoopbackOperator(t *testing.T) {
	gate, err := ui.NewGate(ui.Settings{LoopbackServer: true})
	if err != nil {
		t.Fatal(err)
	}

	policy, err := NewPolicy(PolicyConfig{
		APIKey:    "01234567890123456789012345678901",
		AdminAPIKey: "09876543210987654321098765432109",
		Operator:  gate,
	})
	if err != nil {
		t.Fatal(err)
	}

	probe := httptest.NewRecorder()
	called := false
	handler := policy.Authorize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		metadata, ok := MetadataFrom(r.Context())
		if !ok || metadata.Principal != AdminPrincipalID {
			t.Fatalf("metadata = %+v, ok = %v", metadata, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	request.Host = "127.0.0.1:8080"
	request.RemoteAddr = "127.0.0.1:5000"
	handler.ServeHTTP(probe, request)

	if probe.Code != http.StatusNoContent || !called {
		t.Fatalf("status = %d called = %v", probe.Code, called)
	}
}
