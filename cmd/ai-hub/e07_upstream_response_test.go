package main

import (
	"strings"
	"testing"
)

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
