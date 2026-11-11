package access_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"aihub/internal/access"
)

type admissionProbe struct {
	called      bool
	body        []byte
	contentType string
	session     string
	metadata    access.Metadata
}

func (p *admissionProbe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.called = true
	p.body, _ = io.ReadAll(r.Body)
	p.contentType = r.Header.Get("Content-Type")
	p.session = r.Header.Get("X-Opencode-Session")
	p.metadata, _ = access.MetadataFrom(r.Context())
	w.WriteHeader(http.StatusOK)
}

func admissionRequest(target, body string, headers map[string][]string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	request.Host = "localhost"

	for name, values := range headers {
		request.Header[name] = values
	}

	return request
}

func runAdmission(t *testing.T, admission access.Admission, request *http.Request) (*httptest.ResponseRecorder, *admissionProbe) {
	t.Helper()

	probe := &admissionProbe{}
	recorder := httptest.NewRecorder()
	admission.Apply(probe).ServeHTTP(recorder, request)

	return recorder, probe
}

func TestResponsesAdmissionAcceptsExactLimit(t *testing.T) {
	const limit = 1024
	body := strings.Repeat("a", limit)
	request := admissionRequest("/v1/responses", body, map[string][]string{
		"Content-Type": {"application/json"},
	})

	recorder, probe := runAdmission(t, access.ResponsesAdmission(limit), request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if !probe.called {
		t.Fatal("handler must be called for an accepted body")
	}

	if string(probe.body) != body {
		t.Fatal("accepted body bytes must be preserved")
	}
}

func TestResponsesAdmissionRejectsKnownOversize(t *testing.T) {
	const limit = 1024
	request := admissionRequest("/v1/responses", strings.Repeat("a", limit+1), map[string][]string{
		"Content-Type": {"application/json"},
	})

	recorder, probe := runAdmission(t, access.ResponsesAdmission(limit), request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called for an oversized body")
	}

	assertEnvelope(t, recorder, "payload_too_large")
}

func TestResponsesAdmissionRejectsChunkedOversize(t *testing.T) {
	const limit = 1024
	request := admissionRequest("/v1/responses", strings.Repeat("a", limit+1), map[string][]string{
		"Content-Type": {"application/json"},
	})
	request.ContentLength = -1
	request.TransferEncoding = []string{"chunked"}

	recorder, probe := runAdmission(t, access.ResponsesAdmission(limit), request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called for an oversized chunked body")
	}
}

func TestResponsesAdmissionRejectsMissingMediaType(t *testing.T) {
	request := admissionRequest("/v1/responses", `{"model":"x"}`, nil)

	recorder, probe := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called without a JSON media type")
	}

	assertEnvelope(t, recorder, "unsupported_media_type")
}

func TestResponsesAdmissionAcceptsJSONWithCharset(t *testing.T) {
	const contentType = "application/json; charset=utf-8"
	request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
		"Content-Type": {contentType},
	})

	recorder, probe := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if probe.contentType != contentType {
		t.Fatalf("content type = %q, want the original %q", probe.contentType, contentType)
	}
}

func TestResponsesAdmissionRejectsWrongMediaType(t *testing.T) {
	request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
		"Content-Type": {"text/plain"},
	})

	recorder, _ := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", recorder.Code)
	}
}

func TestResponsesAdmissionContentEncoding(t *testing.T) {
	tests := []struct {
		name       string
		encoding   string
		wantStatus int
	}{
		{name: "identity", encoding: "identity", wantStatus: http.StatusOK},
		{name: "gzip", encoding: "gzip", wantStatus: http.StatusUnsupportedMediaType},
		{name: "multiple values", encoding: "identity, identity", wantStatus: http.StatusUnsupportedMediaType},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
				"Content-Type":     {"application/json"},
				"Content-Encoding": {test.encoding},
			})

			recorder, _ := runAdmission(t, access.ResponsesAdmission(1<<20), request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestResponsesAdmissionRejectsDuplicateContentType(t *testing.T) {
	request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
		"Content-Type": {"application/json", "application/json"},
	})

	recorder, probe := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called with duplicate media headers")
	}

	assertEnvelope(t, recorder, "invalid_request")
}

func TestResponsesAdmissionRejectsDuplicateSession(t *testing.T) {
	request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
		"Content-Type":       {"application/json"},
		"X-Opencode-Session": {"ses_one", "ses_two"},
	})

	recorder, _ := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestResponsesAdmissionGeneratesSession(t *testing.T) {
	request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
		"Content-Type": {"application/json"},
	})
	request = request.WithContext(access.WithMetadata(request.Context(), access.Metadata{
		Principal: access.PrincipalID,
		Host:      "localhost",
	}))

	recorder, probe := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	pattern := regexp.MustCompile(`^aihub-ses-[0-9a-f]{32}$`)
	if !pattern.MatchString(probe.session) {
		t.Fatalf("session = %q, want the generated aihub-ses format", probe.session)
	}

	if probe.metadata.ProviderSessionID != probe.session {
		t.Fatalf("metadata session = %q, want %q", probe.metadata.ProviderSessionID, probe.session)
	}

	if probe.metadata.Principal != access.PrincipalID || probe.metadata.Host != "localhost" {
		t.Fatalf("metadata transport fields were lost: %+v", probe.metadata)
	}
}

func TestResponsesAdmissionPreservesSession(t *testing.T) {
	const session = "ses_context-1.2:3"
	request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
		"Content-Type":       {"application/json"},
		"X-Opencode-Session": {session},
	})

	recorder, probe := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if probe.session != session {
		t.Fatalf("session = %q, want %q", probe.session, session)
	}
}

func TestResponsesAdmissionRejectsInvalidSession(t *testing.T) {
	request := admissionRequest("/v1/responses", `{"model":"x"}`, map[string][]string{
		"Content-Type":       {"application/json"},
		"X-Opencode-Session": {"ses/context"},
	})

	recorder, probe := runAdmission(t, access.ResponsesAdmission(1<<20), request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}

	if probe.called {
		t.Fatal("handler must not be called with an invalid session")
	}
}
