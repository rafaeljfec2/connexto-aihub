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

func TestAuthorizeAllowsUnauthenticatedUISessionProbe(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = 's'
	}

	gate, err := ui.NewGate(ui.Settings{
		Passphrase:    "operator-passphrase-minimum-length",
		SessionSecret: secret,
	})
	if err != nil {
		t.Fatal(err)
	}

	policy, err := NewPolicy(PolicyConfig{
		APIKey:               "01234567890123456789012345678901",
		AdminAPIKey:          "09876543210987654321098765432109",
		AllowedHosts:         []string{"localhost", "127.0.0.1"},
		AllowedHostsExplicit: true,
		AllowedOrigins:       []string{"http://localhost:5173"},
		Operator:             gate,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get ui session bypasses auth check", func(t *testing.T) {
		probe := httptest.NewRecorder()
		called := false
		handler := policy.Authorize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/api/ui/session", nil)
		request.Host = "localhost:5173"
		request.RemoteAddr = "172.18.0.3:50000"
		handler.ServeHTTP(probe, request)

		if probe.Code != http.StatusOK || !called {
			t.Fatalf("status = %d called = %v, want 200 and called=true", probe.Code, called)
		}
	})

	t.Run("get ui session still enforces host and origin", func(t *testing.T) {
		probe := httptest.NewRecorder()
		called := false
		handler := policy.Authorize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/api/ui/session", nil)
		request.Host = "localhost:5173"
		request.Header.Set("Origin", "https://evil.example")
		handler.ServeHTTP(probe, request)

		if probe.Code != http.StatusForbidden || called {
			t.Fatalf("status = %d called = %v, want 403 and called=false", probe.Code, called)
		}
	})

	t.Run("protected routes remain unauthorized without cookie", func(t *testing.T) {
		probe := httptest.NewRecorder()
		called := false
		handler := policy.Authorize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/api/resource-policies", nil)
		request.Host = "localhost:5173"
		request.RemoteAddr = "172.18.0.3:50000"
		handler.ServeHTTP(probe, request)

		if probe.Code != http.StatusUnauthorized || called {
			t.Fatalf("status = %d called = %v, want 401 and called=false", probe.Code, called)
		}
	})
}
