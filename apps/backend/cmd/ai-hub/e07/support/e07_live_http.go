package support

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

const E07MaxHTTPResponseBytes = 2 << 20

const E07RecordQueryTimeout = 20 * time.Second

type E07ExecutionRecord struct {
	ExecutionID string          `json:"execution_id"`
	RequestID   *string         `json:"request_id"`
	Provider    string          `json:"provider_id"`
	ModelID     *string         `json:"model_id"`
	Outcome     string          `json:"outcome"`
	Usage       json.RawMessage `json:"usage"`
}

type E07RecordsPage struct {
	Records []E07ExecutionRecord `json:"records"`
}

// e07ExpectedRecord is derived from the E05 terminal observation, never from
// the gateway's HTTP status.
type E07ExpectedRecord struct {
	RequestID   string
	ExecutionID string
	Outcome     string
}

// e07RecordExpectations keeps the captured requests that never produced an E05
// terminal observation separate from the expected records, so correlation
// remains an independent assertion.
type E07RecordExpectations struct {
	Records           map[string]E07ExpectedRecord
	UnmatchedCaptures int
}

func (expectations E07RecordExpectations) Len() int {
	return len(expectations.Records)
}

// e07RecordAssertions reports each record assertion independently. A failure
// in one assertion never prevents the others from running.
type E07RecordAssertions struct {
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

func (assertions E07RecordAssertions) Pass() bool {
	return assertions.ListOK && assertions.CorrelationOK && assertions.OutcomeOK &&
		assertions.SchemaOK && assertions.ByIDOK && assertions.Expected > 0 &&
		assertions.Unmatched == 0
}

// e07RecordReader is the narrow query seam used by the live scenario and by
// deterministic tests.
type E07RecordReader interface {
	List(context.Context) ([]E07ExecutionRecord, error)
	Get(context.Context, string) (E07ExecutionRecord, error)
}

// e07HTTPRecordReader queries the authenticated E06 routes.
type E07HTTPRecordReader struct {
	BaseURL    string
	GatewayKey string
}

func (reader *E07HTTPRecordReader) List(ctx context.Context) ([]E07ExecutionRecord, error) {
	return E07Records(ctx, reader.BaseURL, reader.GatewayKey)
}

func (reader *E07HTTPRecordReader) Get(ctx context.Context, executionID string) (E07ExecutionRecord, error) {
	return E07GetRecord(ctx, reader.BaseURL, reader.GatewayKey, executionID)
}

func E07ReadResponse(response *http.Response) ([]byte, error) {
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, E07MaxHTTPResponseBytes+1))
	if err != nil {
		return nil, errors.New("HTTP response body could not be read")
	}
	if len(body) > E07MaxHTTPResponseBytes {
		return nil, errors.New("HTTP response body exceeded the validation bound")
	}
	return body, nil
}

func E07Records(ctx context.Context, baseURL, gatewayKey string) ([]E07ExecutionRecord, error) {
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
	body, err := E07ReadResponse(response)
	if err != nil {
		return nil, errors.New("records response could not be read")
	}
	var page E07RecordsPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, errors.New("records response was invalid")
	}
	return page.Records, nil
}

// e07GetRecord queries one record by execution identifier. It never depends on
// a successful list query, so the by-ID assertion stays independent.
func E07GetRecord(ctx context.Context, baseURL, gatewayKey, executionID string) (E07ExecutionRecord, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/records/executions/"+executionID, nil)
	if err != nil {
		return E07ExecutionRecord{}, errors.New("single-record query could not be created")
	}
	request.Header.Set("Authorization", "Bearer "+gatewayKey)
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return E07ExecutionRecord{}, errors.New("single-record query failed")
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return E07ExecutionRecord{}, errors.New("single-record query returned an unexpected status")
	}
	body, err := E07ReadResponse(response)
	if err != nil {
		return E07ExecutionRecord{}, errors.New("single-record response could not be read")
	}
	var record E07ExecutionRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return E07ExecutionRecord{}, errors.New("single-record response was invalid")
	}
	return record, nil
}
