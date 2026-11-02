package main

import (
	"crypto/sha256"
	"errors"
	"hash"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type e07ResponseFidelityObservation struct {
	StatusCode   int
	ContentType  string
	BodyDigest   [32]byte
	BodyComplete bool
}

type e07DigestReadCloser struct {
	body   io.ReadCloser
	hasher hash.Hash
	finish func([32]byte, bool)
	once   sync.Once
}

func newE07DigestReadCloser(body io.ReadCloser, finish func([32]byte, bool)) *e07DigestReadCloser {
	return &e07DigestReadCloser{body: body, hasher: sha256.New(), finish: finish}
}

func (body *e07DigestReadCloser) Read(buffer []byte) (int, error) {
	count, err := body.body.Read(buffer)
	if count > 0 {
		_, _ = body.hasher.Write(buffer[:count])
	}
	if err == io.EOF {
		body.complete(true)
	} else if err != nil {
		body.complete(false)
	}
	return count, err
}

func (body *e07DigestReadCloser) Close() error {
	body.complete(false)
	return body.body.Close()
}

func (body *e07DigestReadCloser) complete(complete bool) {
	body.once.Do(func() {
		var digest [32]byte
		if complete {
			copy(digest[:], body.hasher.Sum(nil))
		}
		body.finish(digest, complete)
	})
}

func e07ClientResponseObservation(response *http.Response, body []byte, readErr error) e07ResponseFidelityObservation {
	observation := e07ResponseFidelityObservation{}
	if response != nil {
		observation.StatusCode = response.StatusCode
		observation.ContentType = response.Header.Get("Content-Type")
	}
	if readErr == nil {
		observation.BodyDigest = sha256.Sum256(body)
		observation.BodyComplete = true
	}
	return observation
}

func e07ResponseObservationsMatch(upstream, client e07ResponseFidelityObservation) bool {
	return upstream.BodyComplete && client.BodyComplete &&
		upstream.StatusCode == client.StatusCode &&
		upstream.ContentType == client.ContentType &&
		upstream.BodyDigest == client.BodyDigest
}

func TestE07ResponseObserverComparesOneCompleteResponseAtBothBoundaries(t *testing.T) {
	const body = `{"error":{"type":"invalid_request_error","message":"synthetic"}}`
	base := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadRequest,
			Header: http.Header{"Content-Type": []string{"application/json"}},
			Body:   io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	observer := newE07ProviderObserver(base, "")
	observer.armResponseFidelityObservation()
	request, err := http.NewRequest(http.MethodPost, "http://hub.invalid/v1/responses", nil)
	if err != nil {
		t.Fatal("validation request could not be created")
	}
	response, err := (&http.Client{Transport: observer}).Do(request)
	if err != nil {
		t.Fatal("validation response could not be received")
	}
	responseBody, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	clientObservation := e07ClientResponseObservation(response, responseBody, readErr)
	clear(responseBody)
	upstreamObservation := observer.takeResponseFidelityObservation()

	if !e07ResponseObservationsMatch(upstreamObservation, clientObservation) {
		t.Fatalf("complete response observations did not match: upstream_complete=%t client_complete=%t",
			upstreamObservation.BodyComplete, clientObservation.BodyComplete)
	}
	if got := observer.BudgetSnapshot().Forwarded; got != 1 {
		t.Fatalf("same-response observer forwarded %d calls, want 1", got)
	}
}

func TestE07ResponseObserverDoesNotAcceptIncompleteOrFailedReads(t *testing.T) {
	tests := []struct {
		name string
		body io.ReadCloser
	}{
		{name: "closed before EOF", body: io.NopCloser(strings.NewReader("synthetic"))},
		{name: "read error", body: failingReadCloser{}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			base := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header),
					Body: testCase.body, Request: request}, nil
			})
			observer := newE07ProviderObserver(base, "")
			observer.armResponseFidelityObservation()
			request, err := http.NewRequest(http.MethodGet, "http://provider.invalid/error", nil)
			if err != nil {
				t.Fatal("validation request could not be created")
			}
			response, err := (&http.Client{Transport: observer}).Do(request)
			if err != nil {
				t.Fatal("validation response could not be received")
			}
			if testCase.name == "closed before EOF" {
				buffer := make([]byte, 1)
				_, _ = response.Body.Read(buffer)
			} else {
				_, _ = response.Body.Read(make([]byte, 1))
			}
			_ = response.Body.Close()
			observation := observer.takeResponseFidelityObservation()
			if observation.BodyComplete {
				t.Fatal("an incomplete or failed body read was marked complete")
			}
		})
	}
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) { return 0, errors.New("synthetic read error") }
func (failingReadCloser) Close() error             { return nil }

func TestE07UpstreamErrorEnvelopeClassification(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{name: "generic error envelope", body: `{"error":{"type":"invalid_request_error","message":"unknown model"}}`, want: true},
		{name: "error key as string", body: `{"error":"upstream failure"}`, want: true},
		{name: "non-error body", body: `{"id":"synthetic","object":"response"}`, want: false},
		{name: "null error", body: `{"error":null}`, want: false},
		{name: "empty body", body: ``, want: false},
		{name: "invalid json", body: `not-json`, want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := e07ErrorEnvelopePresent([]byte(testCase.body)); got != testCase.want {
				t.Fatalf("envelope present = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestE07UpstreamProbeRequiresSameResponseFidelity(t *testing.T) {
	observation := e07UpstreamProbeObservation{
		StatusPreserved:      true,
		ContentTypePreserved: true,
		BodyDigestMatches:    true,
		BodyComplete:         true,
		ClientErrorStatus:    true,
		EnvelopePresent:      true,
		ProviderCalls:        1,
	}
	if !observation.PassthroughObserved() {
		t.Fatal("a complete same-response client error was not accepted")
	}
	if !strings.Contains(observation.Summary(), "status_preserved=true") {
		t.Fatal("upstream diagnostic does not report status preservation separately")
	}

	observation.StatusPreserved = false
	if observation.PassthroughObserved() {
		t.Fatal("a gateway-generated status was accepted as upstream passthrough")
	}

	observation.StatusPreserved = true
	observation.ContentTypePreserved = false
	if observation.PassthroughObserved() {
		t.Fatal("a changed content type was accepted as same-response passthrough")
	}

	observation.ContentTypePreserved = true
	observation.BodyDigestMatches = false
	if observation.PassthroughObserved() {
		t.Fatal("a changed body digest was accepted as same-response passthrough")
	}

	observation.BodyDigestMatches = true
	observation.BodyComplete = false
	if observation.PassthroughObserved() {
		t.Fatal("an incomplete client body was accepted as same-response passthrough")
	}

	observation.BodyComplete = true
	observation.ProviderCalls = 2
	if observation.PassthroughObserved() {
		t.Fatal("multiple provider calls were accepted for the single-call scenario")
	}

	observation.ProviderCalls = 1
	observation.EnvelopePresent = false
	if observation.PassthroughObserved() {
		t.Fatal("a non-envelope body was accepted as upstream passthrough")
	}
	if strings.Contains(observation.Summary(), "body=") {
		t.Fatal("upstream diagnostic summary exposes body content")
	}
}
