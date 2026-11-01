package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"aihub/internal/execution"
)

func TestE07RecordVerificationFollowsE05TerminalSemantics(t *testing.T) {
	cases := []struct {
		name   string
		status int
		opener func() (*http.Response, error)
	}{
		{
			name:   "upstream 2xx fully read",
			status: http.StatusOK,
			opener: func() (*http.Response, error) {
				return e07FixedResponse(http.StatusOK, "synthetic success body"), nil
			},
		},
		{
			name:   "upstream 4xx fully read",
			status: http.StatusBadRequest,
			opener: func() (*http.Response, error) {
				return e07FixedResponse(http.StatusBadRequest, `{"error":{"type":"invalid_request_error"}}`), nil
			},
		},
		{
			name:   "upstream 401 fully read",
			status: http.StatusUnauthorized,
			opener: func() (*http.Response, error) {
				return e07FixedResponse(http.StatusUnauthorized, `{"error":{"type":"authentication_error"}}`), nil
			},
		},
		{
			name:   "transport failure",
			status: http.StatusBadGateway,
			opener: func() (*http.Response, error) {
				return nil, errors.New("synthetic transport failure")
			},
		},
		{
			name:   "body read failure",
			status: http.StatusOK,
			opener: func() (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       e07FailingBody{},
				}, nil
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := e07StartDeterministicRecords(t, func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
				return testCase.opener()
			})
			requestID := "corr_semantic_" + strings.ReplaceAll(testCase.name, " ", "_")
			fixture.execute(t, e07ValidationCommand(requestID))

			expectations := e07BuildExpectedRecords(
				[]e07RequestCapture{{ID: requestID, Status: testCase.status}},
				fixture.recorder.Snapshot(),
				nil,
			)
			assertions := e07AssertRecords(context.Background(), fixture.reader, expectations)

			if !assertions.Pass() {
				t.Fatalf("record assertions failed: %s", assertions.Summary("deterministic"))
			}
			if assertions.Matched != 1 || assertions.ByIDMatched != 1 {
				t.Fatalf("record assertions did not correlate the execution: %s", assertions.Summary("deterministic"))
			}
		})
	}
}
