package access_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/access"
)

func TestAdmissionZeroValuePassesThrough(t *testing.T) {
	probe := &admissionProbe{}
	request := admissionRequest("/v1/responses", "not-json", map[string][]string{
		"Content-Type": {"text/plain"},
	})

	recorder := httptest.NewRecorder()
	access.Admission{}.Apply(probe).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !probe.called {
		t.Fatal("the zero-value admission must not restrict the handler")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingReader) Close() error             { return nil }

func TestAdmissionReadError(t *testing.T) {
	request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
		"Content-Type": {"application/json"},
	})
	request.Body = failingReader{}

	recorder, probe := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called after a body read error")
	}
}

func TestAdmissionDoesNotParseJSON(t *testing.T) {
	const body = "{not valid json"
	request := admissionRequest("/v1/responses", body, map[string][]string{
		"Content-Type": {"application/json"},
	})

	recorder, probe := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if string(probe.body) != body {
		t.Fatal("admission must forward invalid JSON bytes unchanged")
	}
}
