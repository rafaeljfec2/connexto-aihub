package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

// e07CapturingTransport returns one fixed upstream response and retains the
// forwarded request bytes in memory for exactness assertions.
type e07CapturingTransport struct {
	status      int
	contentType string
	body        []byte
	forwarded   []byte
}

func (transport *e07CapturingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Body != nil {
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		transport.forwarded = payload
		_ = request.Body.Close()
	}

	return &http.Response{
		StatusCode: transport.status,
		Header:     http.Header{"Content-Type": []string{transport.contentType}},
		Body:       io.NopCloser(bytes.NewReader(transport.body)),
		Request:    request,
	}, nil
}

func TestE07GatewayPreservesFixedUpstreamResponseDeterministically(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		body        string
		contentType string
	}{
		{
			name:        "client error envelope",
			status:      http.StatusBadRequest,
			contentType: "application/json",
			body:        `{"error":{"type":"invalid_request_error","code":"model_not_found","message":"unknown model"}}`,
		},
		{
			name:        "rate limit envelope",
			status:      http.StatusTooManyRequests,
			contentType: "application/json",
			body:        `{"error":{"type":"rate_limit_error","message":"synthetic upstream limit"}}`,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			payload := []byte(`{"model":"e07-intentionally-unknown-model","input":"Return one short response."}`)
			transport := &e07CapturingTransport{
				status:      testCase.status,
				contentType: "application/json",
				body:        []byte(testCase.body),
			}
			client, err := providers.NewClient(config.ProviderOpenCodeGo{
				BaseURL: "http://provider.invalid",
				APIKey:  "validation-provider-key",
			}, &http.Client{Transport: transport})
			if err != nil {
				t.Fatal("deterministic provider client could not be created")
			}
			handler := gateway.NewResponsesHandler(client, e07DiscardLogger(), time.Minute)

			request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/v1/responses", bytes.NewReader(payload))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.status {
				t.Fatalf("gateway status = %d, want %d", recorder.Code, testCase.status)
			}
			if !bytes.Equal(recorder.Body.Bytes(), []byte(testCase.body)) {
				t.Fatal("gateway transformed the fixed upstream response bytes")
			}
			if got := recorder.Header().Get("Content-Type"); got != testCase.contentType {
				t.Fatalf("gateway content type = %q, want %q", got, testCase.contentType)
			}
			if !bytes.Equal(transport.forwarded, payload) {
				t.Fatal("gateway transformed the forwarded request bytes")
			}
		})
	}
}
