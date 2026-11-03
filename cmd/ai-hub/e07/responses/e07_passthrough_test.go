package responses

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestE07GatewayPreservesFixedUpstreamResponseDeterministically(t *testing.T) {
	cases := []struct {
		name        string
		Status      int
		Body        string
		ContentType string
	}{
		{
			name:        "client error envelope",
			Status:      http.StatusBadRequest,
			ContentType: "application/json",
			Body:        `{"error":{"type":"invalid_request_error","code":"model_not_found","message":"unknown model"}}`,
		},
		{
			name:        "rate limit envelope",
			Status:      http.StatusTooManyRequests,
			ContentType: "application/json",
			Body:        `{"error":{"type":"rate_limit_error","message":"synthetic upstream limit"}}`,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			payload := []byte(`{"model":"e07-intentionally-unknown-model","input":"Return one short response."}`)
			transport := &e07support.E07CapturingTransport{
				Status:      testCase.Status,
				ContentType: "application/json",
				Body:        []byte(testCase.Body),
			}
			client, err := providers.NewClient(config.ProviderOpenCodeGo{
				BaseURL: "http://provider.invalid",
				APIKey:  "validation-provider-key",
			}, &http.Client{Transport: transport})
			if err != nil {
				t.Fatal("deterministic provider client could not be created")
			}
			handler := gateway.NewResponsesHandler(client, e07support.E07DiscardLogger(), time.Minute, nil)

			request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/v1/responses", bytes.NewReader(payload))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.Status {
				t.Fatalf("gateway status = %d, want %d", recorder.Code, testCase.Status)
			}
			if !bytes.Equal(recorder.Body.Bytes(), []byte(testCase.Body)) {
				t.Fatal("gateway transformed the fixed upstream response bytes")
			}
			if got := recorder.Header().Get("Content-Type"); got != testCase.ContentType {
				t.Fatalf("gateway content type = %q, want %q", got, testCase.ContentType)
			}
			if !bytes.Equal(transport.Forwarded, payload) {
				t.Fatal("gateway transformed the forwarded request bytes")
			}
		})
	}
}
