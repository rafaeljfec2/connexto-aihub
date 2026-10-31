package access

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type sessionFailureProbe struct {
	called bool
}

func (p *sessionFailureProbe) ServeHTTP(http.ResponseWriter, *http.Request) {
	p.called = true
}

func TestResponsesAdmissionHandlesSessionGenerationFailure(t *testing.T) {
	admission := Admission{
		kind:  admissionResponses,
		limit: 1 << 20,
		newSession: func() (string, error) {
			return "", errors.New("entropy unavailable")
		},
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"x"}`))
	request.Host = "localhost"
	request.Header.Set("Content-Type", "application/json")

	probe := &sessionFailureProbe{}
	recorder := httptest.NewRecorder()
	admission.Apply(probe).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called when session generation fails")
	}

	if !strings.Contains(recorder.Body.String(), `"type":"internal_error"`) {
		t.Fatalf("body = %q, want an internal_error envelope", recorder.Body.String())
	}

	if strings.Contains(recorder.Body.String(), "entropy unavailable") {
		t.Fatal("internal failure detail must not be exposed")
	}
}
