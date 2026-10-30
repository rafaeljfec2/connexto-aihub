package providers_test

import (
	"io"
	"net/http"
)

// upstreamCapture records the attributes the adapter must control. Pointers
// are used so every fake upstream can be wired to the same builder.
type upstreamCapture struct {
	method      *string
	path        *string
	auth        *string
	session     *string
	contentType *string
	body        *string
}

// sinkHandler returns a 200 handler that records the incoming request and
// answers with a minimal valid Responses body.
func sinkHandler(capture *upstreamCapture) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture.method != nil {
			*capture.method = r.Method
		}
		if capture.path != nil {
			*capture.path = r.URL.Path
		}
		if capture.auth != nil {
			*capture.auth = r.Header.Get("Authorization")
		}
		if capture.session != nil {
			*capture.session = r.Header.Get("X-Opencode-Session")
		}
		if capture.contentType != nil {
			*capture.contentType = r.Header.Get("Content-Type")
		}
		if capture.body != nil {
			*capture.body = readAllString(r.Body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_fixture","object":"response","status":"completed"}`))
	})
}

func readAllString(body io.Reader) string {
	content, err := io.ReadAll(body)
	if err != nil {
		return "read error: " + err.Error()
	}

	return string(content)
}
