package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/routing"
)

// headerRecorder implements http.ResponseWriter for the mapping tests.
type headerRecorder struct {
	status int
	header http.Header
	body   bytes.Buffer
}

func newHeaderRecorder() *headerRecorder {
	return &headerRecorder{header: make(http.Header)}
}

func (r *headerRecorder) Header() http.Header { return r.header }

func (r *headerRecorder) Write(p []byte) (int, error) { return r.body.Write(p) }

func (r *headerRecorder) WriteHeader(status int) { r.status = status }

func (r *headerRecorder) snapshot() (int, http.Header, string) {
	return r.status, r.header, r.body.String()
}

type quotaFailurePayload struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		ResetAt string `json:"reset_at"`
	} `json:"error"`
}

func decodeFailurePayload(t *testing.T, body string) quotaFailurePayload {
	t.Helper()
	var payload quotaFailurePayload
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("error envelope did not parse: %v (body %q)", err, body)
	}
	return payload
}

func quotaResult(cause execution.Cause, resetIn *int64) execution.Result {
	return execution.Result{
		ExecutionID:    "0f959d684e58238ca2167225fba52143",
		Outcome:        execution.OutcomeFailed,
		Phase:          execution.PhaseOpening,
		Cause:          cause,
		QuotaResetInNS: resetIn,
	}
}

func TestWriteExecutionFailureResponseMapsUpstreamQuotaTo429(t *testing.T) {
	resetIn := int64((54 * time.Hour).Nanoseconds())
	recorder := newHeaderRecorder()
	writeExecutionFailureResponse(recorder, quotaResult(execution.CauseUpstreamQuota, &resetIn), routing.Target{})

	status, headers, body := recorder.snapshot()
	if status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", status)
	}
	if got := headers.Get("Content-Type"); got == "" {
		t.Fatalf("missing content-type header")
	}
	payload := decodeFailurePayload(t, body)
	if payload.Error.Type != "upstream_quota_exceeded" {
		t.Fatalf("error type = %q, want upstream_quota_exceeded", payload.Error.Type)
	}
	if payload.Error.Message == "" {
		t.Fatalf("message empty, want sanitized generic message")
	}
	// The upstream notice text must never surface in the response body.
	if bytes.Contains([]byte(body), []byte("Individual quota reached")) {
		t.Fatalf("body leaks the upstream notice text: %q", body)
	}
	resetAt, parseErr := time.Parse(time.RFC3339, payload.Error.ResetAt)
	if parseErr != nil {
		t.Fatalf("reset_at unparsable: %v", parseErr)
	}
	until := time.Until(resetAt)
	if until < 53*time.Hour+59*time.Minute || until > 54*time.Hour+1*time.Minute {
		t.Fatalf("reset_at = %v, want ~54h ahead", resetAt)
	}
}

func TestWriteExecutionFailureResponseKeepsHistoricalMappings(t *testing.T) {
	t.Run("transport failure keeps 502", func(t *testing.T) {
		recorder := newHeaderRecorder()
		writeExecutionFailureResponse(recorder, quotaResult(execution.CauseTransport, nil), routing.Target{})
		if status, _, _ := recorder.snapshot(); status != http.StatusBadGateway {
			t.Fatalf("status = %d, want 502", status)
		}
	})

	t.Run("opening timeout keeps 504", func(t *testing.T) {
		recorder := newHeaderRecorder()
		result := quotaResult(execution.CauseOpening, nil)
		result.Outcome = execution.OutcomeTimedOut
		writeExecutionFailureResponse(recorder, result, routing.Target{})
		if status, _, _ := recorder.snapshot(); status != http.StatusGatewayTimeout {
			t.Fatalf("status = %d, want 504", status)
		}
	})

	t.Run("quota without reset window answers 429 without reset_at", func(t *testing.T) {
		recorder := newHeaderRecorder()
		writeExecutionFailureResponse(recorder, quotaResult(execution.CauseUpstreamQuota, nil), routing.Target{})
		status, _, body := recorder.snapshot()
		if status != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", status)
		}
		payload := decodeFailurePayload(t, body)
		if payload.Error.ResetAt != "" {
			t.Fatalf("reset_at = %q, want empty", payload.Error.ResetAt)
		}
	})
}
