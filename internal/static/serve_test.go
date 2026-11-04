package static

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIPrefix(t *testing.T) {
	cases := map[string]bool{
		"/healthz":                    true,
		"/readyz":                     true,
		"/v1/models":                  true,
		"/api/events":                 true,
		"/diagnostics/providers":      true,
		"/":                           false,
		"/executions":                 false,
		"/assets/index.js":            false,
	}
	for path, want := range cases {
		if got := APIPrefix(path); got != want {
			t.Fatalf("APIPrefix(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestWrapRoutesAPIBeforeStatic(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	static := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Wrap(api, static)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("api status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/overview", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("static status = %d", rec.Code)
	}
}

func TestHandlerServesEmbeddedIndex(t *testing.T) {
	handler, err := Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v (sync frontend dist before tests)", err)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<") {
		t.Fatal("expected html body")
	}
}
