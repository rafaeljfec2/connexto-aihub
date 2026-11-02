package support

import (
	"bytes"
	"io"
	"net/http"
)

// e07CapturingTransport returns one fixed upstream response and retains the
// forwarded request bytes in memory for exactness assertions.
type E07CapturingTransport struct {
	Status      int
	ContentType string
	Body        []byte
	Forwarded   []byte
}

func (transport *E07CapturingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Body != nil {
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		transport.Forwarded = payload
		_ = request.Body.Close()
	}

	return &http.Response{
		StatusCode: transport.Status,
		Header:     http.Header{"Content-Type": []string{transport.ContentType}},
		Body:       io.NopCloser(bytes.NewReader(transport.Body)),
		Request:    request,
	}, nil
}
