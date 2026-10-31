package gateway_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/records"
	"aihub/internal/server"
)

const recordsRequestID = "req-0123456789abcdef"

func newRecordsService(t *testing.T, start bool) *records.Service {
	t.Helper()

	service := records.New(records.Options{
		Dir:               t.TempDir(),
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxAge:            720 * time.Hour,
		MaxCount:          10000,
		MaxStorageBytes:   64 << 20,
		SegmentBytes:      1 << 20,
		RetentionInterval: time.Hour,
	})
	if start {
		if err := service.Start(context.Background()); err != nil {
			t.Fatalf("records Start() returned error: %v", err)
		}
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	return service
}

func recordsExecutionID(n int) string {
	return fmt.Sprintf("%032x", n)
}

func recordsTime(second int) time.Time {
	return time.Date(2026, 10, 31, 12, 0, second, 123456789, time.UTC)
}

func seedExecution(t *testing.T, service *records.Service, id string, completedAt time.Time, outcome execution.Outcome, providerID, modelID string) {
	t.Helper()

	const duration = time.Second
	cause := execution.Cause("")
	if outcome != execution.OutcomeCompleted {
		cause = execution.CauseTransport
	}
	status := 200
	if outcome != execution.OutcomeCompleted {
		status = 0
	}

	service.OnExecutionFinished(execution.FinishedExecution{
		Result: execution.Result{
			ExecutionID:    id,
			Outcome:        outcome,
			Phase:          execution.PhaseForwarding,
			Cause:          cause,
			UpstreamStatus: status,
			Committed:      true,
			Duration:       duration,
		},
		RequestID: recordsRequestID,
		Target:    execution.Target{ProviderID: providerID, ModelID: modelID},
		StartedAt: completedAt.Add(-duration).UTC(),
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok, err := service.Get(id); err == nil && ok {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("record %s was not indexed before the timeout", id)
}

func newRecordsRouter(t *testing.T, service *records.Service) http.Handler {
	t.Helper()

	logger, _ := newTestLogger()

	return server.NewRouter(
		logger,
		admissionGatewayPolicy(t),
		health.Live(),
		health.NewRegistry().Ready(),
		server.Route{
			Method:  http.MethodGet,
			Pattern: "/v1/records/executions",
			Handler: gateway.NewExecutionRecordsHandler(service),
		},
		server.Route{
			Method:  http.MethodGet,
			Pattern: "/v1/records/executions/{execution_id}",
			Handler: gateway.NewExecutionRecordHandler(service),
		},
	)
}

func recordsGet(t *testing.T, router http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.Header.Set("Authorization", "Bearer "+gatewayClientKey)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func TestRecordsRoutesRequireE04Admission(t *testing.T) {
	router := newRecordsRouter(t, newRecordsService(t, true))

	for _, target := range []string{"/v1/records/executions", "/v1/records/executions/" + recordsExecutionID(1)} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s without bearer = %d, want 401", target, recorder.Code)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/records/executions", nil)
	request.Host = "evil.test"
	request.Header.Set("Authorization", "Bearer "+gatewayClientKey)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("disallowed host = %d, want 403", recorder.Code)
	}
}

func TestExecutionRecordLookupExposesOnlySchemaFields(t *testing.T) {
	service := newRecordsService(t, true)
	seedExecution(t, service, recordsExecutionID(1), recordsTime(1), execution.OutcomeCompleted, "opencode-go", "vendor/model")
	router := newRecordsRouter(t, service)

	recorder := recordsGet(t, router, "/v1/records/executions/"+recordsExecutionID(1))
	if recorder.Code != http.StatusOK {
		t.Fatalf("lookup status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	want := []string{
		"schema_version", "execution_id", "request_id", "provider_id", "model_id",
		"started_at", "completed_at", "duration_ns", "outcome", "phase", "cause",
		"upstream_status", "response_committed", "usage",
	}
	if len(payload) != len(want) {
		t.Fatalf("response fields = %d, want exactly %d: %v", len(payload), len(want), payload)
	}
	for _, field := range want {
		if _, ok := payload[field]; !ok {
			t.Fatalf("response is missing %s", field)
		}
	}
	if string(payload["usage"]) != "null" {
		t.Fatalf("usage = %s, want null", payload["usage"])
	}
	if string(payload["model_id"]) != `"vendor/model"` {
		t.Fatalf("model_id = %s, want vendor/model", payload["model_id"])
	}

	missing := recordsGet(t, router, "/v1/records/executions/"+recordsExecutionID(99))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing record status = %d, want 404", missing.Code)
	}
	if failureType(t, missing.Body.Bytes()) != "not_found" {
		t.Fatalf("missing record failure type = %q, want not_found", failureType(t, missing.Body.Bytes()))
	}

	malformed := recordsGet(t, router, "/v1/records/executions/not-an-execution-id")
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed id status = %d, want 400", malformed.Code)
	}
	if failureType(t, malformed.Body.Bytes()) != "invalid_request" {
		t.Fatalf("malformed id failure type = %q, want invalid_request", failureType(t, malformed.Body.Bytes()))
	}
}

func TestExecutionRecordsReportDegradedHistory(t *testing.T) {
	service := newRecordsService(t, true)
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
	service.OnExecutionFinished(execution.FinishedExecution{
		Result:    execution.Result{ExecutionID: recordsExecutionID(1), Outcome: execution.OutcomeCompleted, Phase: execution.PhaseForwarding, Duration: time.Second},
		RequestID: recordsRequestID,
		Target:    execution.Target{ProviderID: "opencode-go"},
		StartedAt: recordsTime(1).Add(-time.Second),
	})

	router := newRecordsRouter(t, service)
	page := decodeRecordsList(t, recordsGet(t, router, "/v1/records/executions"))
	if !page.Degraded {
		t.Fatal("list response must report process-local degradation")
	}
}

func TestExecutionRecordsUnavailableIndex(t *testing.T) {
	router := newRecordsRouter(t, newRecordsService(t, false))

	for _, target := range []string{"/v1/records/executions", "/v1/records/executions/" + recordsExecutionID(1)} {
		recorder := recordsGet(t, router, target)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s status = %d, want 503", target, recorder.Code)
		}
		if failureType(t, recorder.Body.Bytes()) != "records_unavailable" {
			t.Fatalf("%s failure type = %q, want records_unavailable", target, failureType(t, recorder.Body.Bytes()))
		}
	}
}

func failureType(t *testing.T, body []byte) string {
	t.Helper()

	var envelope struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}

	return envelope.Error.Type
}
