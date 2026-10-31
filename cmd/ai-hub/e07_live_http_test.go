package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"aihub/internal/providers"
)

const e07MaxHTTPResponseBytes = 2 << 20

const e07RecordQueryTimeout = 20 * time.Second

type e07ExecutionRecord struct {
	ExecutionID string          `json:"execution_id"`
	RequestID   *string         `json:"request_id"`
	Provider    string          `json:"provider_id"`
	ModelID     *string         `json:"model_id"`
	Outcome     string          `json:"outcome"`
	Usage       json.RawMessage `json:"usage"`
}

type e07RecordsPage struct {
	Records []e07ExecutionRecord `json:"records"`
}

// e07ExpectedRecord is derived from the E05 terminal observation, never from
// the gateway's HTTP status.
type e07ExpectedRecord struct {
	RequestID   string
	ExecutionID string
	Outcome     string
}

// e07RecordExpectations keeps the captured requests that never produced an E05
// terminal observation separate from the expected records, so correlation
// remains an independent assertion.
type e07RecordExpectations struct {
	Records           map[string]e07ExpectedRecord
	UnmatchedCaptures int
}

func (expectations e07RecordExpectations) Len() int {
	return len(expectations.Records)
}

// e07RecordAssertions reports each record assertion independently. A failure
// in one assertion never prevents the others from running.
type e07RecordAssertions struct {
	ListOK        bool
	CorrelationOK bool
	OutcomeOK     bool
	SchemaOK      bool
	ByIDOK        bool
	Expected      int
	Matched       int
	ByIDRequested int
	ByIDMatched   int
	Unmatched     int
}

func (assertions e07RecordAssertions) Pass() bool {
	return assertions.ListOK && assertions.CorrelationOK && assertions.OutcomeOK &&
		assertions.SchemaOK && assertions.ByIDOK && assertions.Expected > 0 &&
		assertions.Unmatched == 0
}

// e07RecordReader is the narrow query seam used by the live scenario and by
// deterministic tests.
type e07RecordReader interface {
	List(context.Context) ([]e07ExecutionRecord, error)
	Get(context.Context, string) (e07ExecutionRecord, error)
}

// e07HTTPRecordReader queries the authenticated E06 routes.
type e07HTTPRecordReader struct {
	baseURL    string
	gatewayKey string
}

func (reader *e07HTTPRecordReader) List(ctx context.Context) ([]e07ExecutionRecord, error) {
	return e07Records(ctx, reader.baseURL, reader.gatewayKey)
}

func (reader *e07HTTPRecordReader) Get(ctx context.Context, executionID string) (e07ExecutionRecord, error) {
	return e07GetRecord(ctx, reader.baseURL, reader.gatewayKey, executionID)
}

func e07ReadResponse(response *http.Response) ([]byte, error) {
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, e07MaxHTTPResponseBytes+1))
	if err != nil {
		return nil, errors.New("HTTP response body could not be read")
	}
	if len(body) > e07MaxHTTPResponseBytes {
		return nil, errors.New("HTTP response body exceeded the validation bound")
	}
	return body, nil
}

func e07Records(ctx context.Context, baseURL, gatewayKey string) ([]e07ExecutionRecord, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/records/executions?limit=200", nil)
	if err != nil {
		return nil, errors.New("records query could not be created")
	}
	request.Header.Set("Authorization", "Bearer "+gatewayKey)
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return nil, errors.New("records query failed")
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, errors.New("records query returned an unexpected status")
	}
	body, err := e07ReadResponse(response)
	if err != nil {
		return nil, errors.New("records response could not be read")
	}
	var page e07RecordsPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, errors.New("records response was invalid")
	}
	return page.Records, nil
}

// e07GetRecord queries one record by execution identifier. It never depends on
// a successful list query, so the by-ID assertion stays independent.
func e07GetRecord(ctx context.Context, baseURL, gatewayKey, executionID string) (e07ExecutionRecord, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/records/executions/"+executionID, nil)
	if err != nil {
		return e07ExecutionRecord{}, errors.New("single-record query could not be created")
	}
	request.Header.Set("Authorization", "Bearer "+gatewayKey)
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return e07ExecutionRecord{}, errors.New("single-record query failed")
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return e07ExecutionRecord{}, errors.New("single-record query returned an unexpected status")
	}
	body, err := e07ReadResponse(response)
	if err != nil {
		return e07ExecutionRecord{}, errors.New("single-record response could not be read")
	}
	var record e07ExecutionRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return e07ExecutionRecord{}, errors.New("single-record response was invalid")
	}
	return record, nil
}

// e07BuildExpectedRecords joins captured gateway requests with the E05 terminal
// observations recorded by the validation observer. The gateway status is
// irrelevant to the expected outcome; a fully read upstream 4xx response is a
// completed E05 execution, while a transport or body-read failure is not.
// Requests rejected before E05 admission are excluded by identifier (the
// authentication probe), never by status.
func e07BuildExpectedRecords(captures []e07RequestCapture, terminals []e07TerminalObservation, preExecution []string) e07RecordExpectations {
	byRequest := make(map[string]e07TerminalObservation, len(terminals))
	for _, terminal := range terminals {
		if terminal.RequestID == "" || terminal.ExecutionID == "" || terminal.Outcome == "" {
			continue
		}
		byRequest[terminal.RequestID] = terminal
	}
	ignored := make(map[string]struct{}, len(preExecution))
	for _, requestID := range preExecution {
		if requestID != "" {
			ignored[requestID] = struct{}{}
		}
	}

	expectations := e07RecordExpectations{Records: make(map[string]e07ExpectedRecord, len(terminals))}
	seen := make(map[string]struct{}, len(captures))
	for _, capture := range captures {
		if capture.ID == "" {
			continue
		}
		if _, excluded := ignored[capture.ID]; excluded {
			continue
		}
		if _, duplicate := seen[capture.ID]; duplicate {
			continue
		}
		seen[capture.ID] = struct{}{}

		terminal, observed := byRequest[capture.ID]
		if !observed {
			expectations.UnmatchedCaptures++

			continue
		}
		expectations.Records[terminal.ExecutionID] = e07ExpectedRecord{
			RequestID:   capture.ID,
			ExecutionID: terminal.ExecutionID,
			Outcome:     terminal.Outcome,
		}
	}

	return expectations
}

// e07AssertRecords evaluates list, correlation, outcome, schema, and by-ID
// assertions independently. The by-ID query runs even when the list query or a
// list-based assertion fails.
func e07AssertRecords(ctx context.Context, reader e07RecordReader, expectations e07RecordExpectations) e07RecordAssertions {
	return e07AssertRecordsWithin(ctx, reader, expectations, time.Time{})
}

// e07AssertRecordsWithin bounds every query by the remaining poll window as well
// as the per-query timeout.
func e07AssertRecordsWithin(ctx context.Context, reader e07RecordReader, expectations e07RecordExpectations, deadline time.Time) e07RecordAssertions {
	assertions := e07RecordAssertions{Expected: expectations.Len(), Unmatched: expectations.UnmatchedCaptures}

	listContext, cancelList := e07RecordQueryContext(ctx, deadline)
	records, listErr := reader.List(listContext)
	cancelList()
	if listErr == nil {
		assertions.ListOK = true
		assertions.CorrelationOK = expectations.Len() > 0
		assertions.OutcomeOK = expectations.Len() > 0
		assertions.SchemaOK = expectations.Len() > 0

		listed := make(map[string]e07ExecutionRecord, len(records))
		for _, record := range records {
			listed[record.ExecutionID] = record
		}
		for executionID, expected := range expectations.Records {
			record, found := listed[executionID]
			if !found {
				assertions.CorrelationOK = false
				assertions.OutcomeOK = false
				assertions.SchemaOK = false

				continue
			}
			assertions.Matched++
			if record.RequestID == nil || *record.RequestID != expected.RequestID {
				assertions.CorrelationOK = false
			}
			if record.Outcome != expected.Outcome {
				assertions.OutcomeOK = false
			}
			if !e07RecordSchemaValid(record) {
				assertions.SchemaOK = false
			}
		}
	}

	assertions.ByIDOK = expectations.Len() > 0
	for executionID, expected := range expectations.Records {
		assertions.ByIDRequested++
		getContext, cancelGet := e07RecordQueryContext(ctx, deadline)
		record, err := reader.Get(getContext, executionID)
		cancelGet()
		if err != nil || !e07RecordMatchesExpected(record, expected) {
			assertions.ByIDOK = false

			continue
		}
		assertions.ByIDMatched++
	}

	return assertions
}

// e07RecordQueryContext gives every query its own bounded context, so a slow
// list query cannot exhaust the by-ID budget. When a poll deadline is present,
// the per-query timeout is clamped to the remaining budget.
func e07RecordQueryContext(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	timeout := e07RecordQueryTimeout
	if !deadline.IsZero() {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			remaining = time.Millisecond
		}
		if remaining < timeout {
			timeout = remaining
		}
	}

	return context.WithTimeout(context.WithoutCancel(parent), timeout)
}

func e07RecordMatchesExpected(record e07ExecutionRecord, expected e07ExpectedRecord) bool {
	return record.ExecutionID == expected.ExecutionID && record.RequestID != nil &&
		*record.RequestID == expected.RequestID && record.Outcome == expected.Outcome &&
		e07RecordSchemaValid(record)
}

func e07RecordSchemaValid(record e07ExecutionRecord) bool {
	return record.Provider == providers.OpenCodeGoID && record.ModelID == nil &&
		bytes.Equal(record.Usage, []byte("null"))
}

// e07AssertRecordsEventually polls within the caller's window so a record that
// is still being written does not produce an immediate false failure.
func e07AssertRecordsEventually(ctx context.Context, reader e07RecordReader, expectations e07RecordExpectations, window time.Duration) e07RecordAssertions {
	deadline := time.Now().Add(window)
	for {
		assertions := e07AssertRecordsWithin(ctx, reader, expectations, deadline)
		if assertions.Pass() || ctx.Err() != nil || time.Now().After(deadline) {
			return assertions
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// e07AssertRecordsAfterRestart always derives a fresh context from the parent's
// values while discarding its deadline and cancellation, so an expired
// pre-stop context cannot suppress the post-restart verification.
func e07AssertRecordsAfterRestart(parent context.Context, reader e07RecordReader, expectations e07RecordExpectations, window time.Duration) e07RecordAssertions {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), window+5*time.Second)
	defer cancel()

	deadline := time.Now().Add(window)
	for {
		assertions := e07AssertRecordsWithin(ctx, reader, expectations, deadline)
		if assertions.Pass() || time.Now().After(deadline) {
			return assertions
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (assertions e07RecordAssertions) Summary(phase string) string {
	return "E07 records phase=" + phase +
		" list=" + e07IntegrityResult(assertions.ListOK) +
		" correlation=" + e07IntegrityResult(assertions.CorrelationOK) +
		" outcome=" + e07IntegrityResult(assertions.OutcomeOK) +
		" schema=" + e07IntegrityResult(assertions.SchemaOK) +
		" by_id=" + e07IntegrityResult(assertions.ByIDOK) +
		" expected=" + strconv.Itoa(assertions.Expected) +
		" matched=" + strconv.Itoa(assertions.Matched) +
		" by_id_requested=" + strconv.Itoa(assertions.ByIDRequested) +
		" by_id_matched=" + strconv.Itoa(assertions.ByIDMatched) +
		" unmatched_captures=" + strconv.Itoa(assertions.Unmatched)
}
