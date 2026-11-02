package responses

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestE07ResponseObserverComparesOneCompleteResponseAtBothBoundaries(t *testing.T) {
	const body = `{"error":{"type":"invalid_request_error","message":"synthetic"}}`
	base := e07support.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadRequest,
			Header: http.Header{"Content-Type": []string{"application/json"}},
			Body:   io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	observer := e07support.NewE07ProviderObserver(base, "")
	observer.ArmResponseFidelityObservation()
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
	clientObservation := e07support.E07ClientResponseObservation(response, responseBody, readErr)
	clear(responseBody)
	upstreamObservation := observer.TakeResponseFidelityObservation()

	if !e07support.E07ResponseObservationsMatch(upstreamObservation, clientObservation) {
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
		Body io.ReadCloser
	}{
		{name: "closed before EOF", Body: io.NopCloser(strings.NewReader("synthetic"))},
		{name: "read error", Body: e07support.FailingReadCloser{}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			base := e07support.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header),
					Body: testCase.Body, Request: request}, nil
			})
			observer := e07support.NewE07ProviderObserver(base, "")
			observer.ArmResponseFidelityObservation()
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
			observation := observer.TakeResponseFidelityObservation()
			if observation.BodyComplete {
				t.Fatal("an incomplete or failed body read was marked complete")
			}
		})
	}
}

func TestE07UpstreamErrorEnvelopeClassification(t *testing.T) {
	cases := []struct {
		name string
		Body string
		want bool
	}{
		{name: "generic error envelope", Body: `{"error":{"type":"invalid_request_error","message":"unknown model"}}`, want: true},
		{name: "error key as string", Body: `{"error":"upstream failure"}`, want: true},
		{name: "non-error body", Body: `{"id":"synthetic","object":"response"}`, want: false},
		{name: "null error", Body: `{"error":null}`, want: false},
		{name: "empty body", Body: ``, want: false},
		{name: "invalid json", Body: `not-json`, want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := e07support.E07ErrorEnvelopePresent([]byte(testCase.Body)); got != testCase.want {
				t.Fatalf("envelope present = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestE07UpstreamProbeRequiresSameResponseFidelity(t *testing.T) {
	observation := e07support.E07UpstreamProbeObservation{
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
