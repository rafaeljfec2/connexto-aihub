package gateway_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
)

func TestExecutionRecordsListPagination(t *testing.T) {
	service := newRecordsService(t, true)
	for n := 1; n <= 5; n++ {
		seedExecution(t, service, recordsExecutionID(n), recordsTime(n), execution.OutcomeCompleted, "opencode-go", "")
	}
	router := newRecordsRouter(t, service)

	first := decodeRecordsList(t, recordsGet(t, router, "/v1/records/executions?limit=2"))
	if first.Count != 2 || len(first.Records) != 2 {
		t.Fatalf("first page = %d records, want 2", first.Count)
	}
	if first.Records[0]["execution_id"] != recordsExecutionID(5) || first.Records[1]["execution_id"] != recordsExecutionID(4) {
		t.Fatalf("first page order = %v, want descending completed_at", first.Records)
	}
	if first.NextCursor == nil {
		t.Fatal("first page must return a next cursor")
	}

	second := decodeRecordsList(t, recordsGet(t, router, "/v1/records/executions?limit=2&cursor="+url.QueryEscape(*first.NextCursor)))
	if second.Records[0]["execution_id"] != recordsExecutionID(3) || second.Records[1]["execution_id"] != recordsExecutionID(2) {
		t.Fatalf("second page order = %v, want the following rows", second.Records)
	}

	third := decodeRecordsList(t, recordsGet(t, router, "/v1/records/executions?limit=2&cursor="+url.QueryEscape(*second.NextCursor)))
	if third.Count != 1 || third.Records[0]["execution_id"] != recordsExecutionID(1) {
		t.Fatalf("third page = %v, want the last row", third.Records)
	}
	if third.NextCursor != nil {
		t.Fatalf("last page next cursor = %q, want null", *third.NextCursor)
	}
}

func TestExecutionRecordsListAppliesFilters(t *testing.T) {
	service := newRecordsService(t, true)
	seedExecution(t, service, recordsExecutionID(1), recordsTime(1), execution.OutcomeCompleted, "opencode-go", "vendor/model")
	seedExecution(t, service, recordsExecutionID(2), recordsTime(2), execution.OutcomeCompleted, "other", "")
	seedExecution(t, service, recordsExecutionID(3), recordsTime(3), execution.OutcomeFailed, "opencode-go", "")
	router := newRecordsRouter(t, service)

	tests := []struct {
		query string
		want  []string
	}{
		{query: "provider_id=opencode-go", want: []string{recordsExecutionID(3), recordsExecutionID(1)}},
		{query: "provider_id=missing", want: nil},
		{query: "model_id=vendor/model", want: []string{recordsExecutionID(1)}},
		{query: "outcome=failed", want: []string{recordsExecutionID(3)}},
		{query: "started_after=" + url.QueryEscape(recordsTime(1).Add(-100*time.Millisecond).Format(time.RFC3339)), want: []string{recordsExecutionID(3), recordsExecutionID(2)}},
		{query: "started_before=" + url.QueryEscape(recordsTime(2).Add(-time.Second).Format(time.RFC3339)), want: []string{recordsExecutionID(1)}},
	}

	for _, test := range tests {
		page := decodeRecordsList(t, recordsGet(t, router, "/v1/records/executions?"+test.query))
		if len(page.Records) != len(test.want) {
			t.Fatalf("%s returned %d records, want %d", test.query, len(page.Records), len(test.want))
		}
		for position, id := range test.want {
			if page.Records[position]["execution_id"] != id {
				t.Fatalf("%s record %d = %v, want %s", test.query, position, page.Records[position]["execution_id"], id)
			}
		}
	}
}

func TestExecutionRecordsListRejectsInvalidParameters(t *testing.T) {
	router := newRecordsRouter(t, newRecordsService(t, true))

	tests := []string{
		"limit=0",
		"limit=201",
		"limit=abc",
		"limit=1&limit=2",
		"unknown=1",
		"outcome=unknown",
		"started_after=not-a-time",
		"started_after=" + url.QueryEscape(recordsTime(2).Format(time.RFC3339)) + "&started_before=" + url.QueryEscape(recordsTime(1).Format(time.RFC3339)),
		"provider_id=",
		"model_id=" + strings.Repeat("m", 4097),
		"cursor=",
		"cursor=%25%25%25",
	}

	for _, query := range tests {
		recorder := recordsGet(t, router, "/v1/records/executions?"+query)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400 (body %s)", query, recorder.Code, recorder.Body.String())
		}
		if failureType(t, recorder.Body.Bytes()) != "invalid_request" {
			t.Fatalf("%s failure type = %q, want invalid_request", query, failureType(t, recorder.Body.Bytes()))
		}
	}
}

type recordsListBody struct {
	Records    []map[string]any `json:"records"`
	Count      int              `json:"count"`
	NextCursor *string          `json:"next_cursor"`
	Degraded   bool             `json:"degraded"`
}

func decodeRecordsList(t *testing.T, recorder *httptest.ResponseRecorder) recordsListBody {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}

	var body recordsListBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list body: %v", err)
	}

	return body
}
