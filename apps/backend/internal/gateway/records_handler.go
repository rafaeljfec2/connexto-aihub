package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"

	"aihub/internal/records"
)

const (
	recordNotFoundType        = "not_found"
	recordsUnavailableType    = "records_unavailable"
	recordNotFoundMessage     = "execution record not found"
	recordsUnavailableMessage = "execution records are unavailable"
	maxProviderFilterBytes    = 128
	maxModelFilterBytes       = 4096
)

type executionRecordDTO struct {
	SchemaVersion     int             `json:"schema_version"`
	ExecutionID       string          `json:"execution_id"`
	RequestID         *string         `json:"request_id"`
	ProviderID        string          `json:"provider_id"`
	ModelID           *string         `json:"model_id"`
	StartedAt         string          `json:"started_at"`
	CompletedAt       string          `json:"completed_at"`
	DurationNS        int64           `json:"duration_ns"`
	Outcome           string          `json:"outcome"`
	Phase             string          `json:"phase"`
	Cause             *string         `json:"cause"`
	UpstreamStatus    *int            `json:"upstream_status"`
	ResponseCommitted bool            `json:"response_committed"`
	Usage             json.RawMessage `json:"usage"`
}

type executionRecordsDTO struct {
	Records    []executionRecordDTO `json:"records"`
	Count      int                  `json:"count"`
	NextCursor *string              `json:"next_cursor"`
	Degraded   bool                 `json:"degraded"`
}

// NewExecutionRecordHandler exposes one retained execution record.
func NewExecutionRecordHandler(service *records.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		id := r.PathValue("execution_id")
		if !records.ValidExecutionID(id) {
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, "invalid execution_id")

			return
		}

		record, ok, err := service.Get(id)
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, recordsUnavailableType, recordsUnavailableMessage)

			return
		}
		if !ok {
			writeJSONError(w, http.StatusNotFound, recordNotFoundType, recordNotFoundMessage)

			return
		}

		writeJSON(w, http.StatusOK, newExecutionRecordDTO(record))
	})
}

// NewExecutionRecordsHandler exposes one filtered page of retained records.
func NewExecutionRecordsHandler(service *records.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		filter, err := parseRecordFilter(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, err.Error())

			return
		}

		page, err := service.List(filter)
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, recordsUnavailableType, recordsUnavailableMessage)

			return
		}

		payload := executionRecordsDTO{
			Records:  make([]executionRecordDTO, 0, len(page.Records)),
			Count:    len(page.Records),
			Degraded: service.Degraded(),
		}
		for _, record := range page.Records {
			payload.Records = append(payload.Records, newExecutionRecordDTO(record))
		}
		if page.NextCursor != "" {
			payload.NextCursor = &page.NextCursor
		}

		writeJSON(w, http.StatusOK, payload)
	})
}

func newExecutionRecordDTO(record records.Record) executionRecordDTO {
	return executionRecordDTO{
		SchemaVersion:     record.SchemaVersion,
		ExecutionID:       record.ExecutionID,
		RequestID:         record.RequestID,
		ProviderID:        record.ProviderID,
		ModelID:           record.ModelID,
		StartedAt:         record.StartedAt.UTC().Format(time.RFC3339Nano),
		CompletedAt:       record.CompletedAt.UTC().Format(time.RFC3339Nano),
		DurationNS:        record.DurationNS,
		Outcome:           record.Outcome,
		Phase:             record.Phase,
		Cause:             record.Cause,
		UpstreamStatus:    record.UpstreamStatus,
		ResponseCommitted: record.ResponseCommitted,
		Usage:             json.RawMessage("null"),
	}
}

func parseRecordFilter(r *http.Request) (records.Filter, error) {
	query := r.URL.Query()
	known := map[string]struct{}{
		"limit": {}, "provider_id": {}, "model_id": {}, "outcome": {},
		"started_after": {}, "started_before": {}, "cursor": {},
	}

	for name, values := range query {
		if _, ok := known[name]; !ok {
			return records.Filter{}, errors.New("unknown query parameter")
		}
		if len(values) > 1 {
			return records.Filter{}, errors.New("query parameter must appear at most once")
		}
	}

	filter := records.Filter{Limit: records.DefaultPageLimit}

	if query.Has("limit") {
		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil || limit < 1 || limit > records.MaxPageLimit {
			return records.Filter{}, errors.New("limit must be between 1 and 200")
		}
		filter.Limit = limit
	}
	if query.Has("provider_id") {
		value := query.Get("provider_id")
		if !validFilterValue(value, maxProviderFilterBytes) {
			return records.Filter{}, errors.New("provider_id is outside its byte bounds")
		}
		filter.ProviderID = value
	}
	if query.Has("model_id") {
		value := query.Get("model_id")
		if !validFilterValue(value, maxModelFilterBytes) {
			return records.Filter{}, errors.New("model_id is outside its byte bounds")
		}
		filter.ModelID = value
	}
	if query.Has("outcome") {
		if !knownRecordOutcome(query.Get("outcome")) {
			return records.Filter{}, errors.New("outcome is not supported")
		}
		filter.Outcome = query.Get("outcome")
	}

	after, err := parseFilterTimestamp(query, "started_after")
	if err != nil {
		return records.Filter{}, err
	}
	before, err := parseFilterTimestamp(query, "started_before")
	if err != nil {
		return records.Filter{}, err
	}
	if after != nil && before != nil && !after.Before(*before) {
		return records.Filter{}, errors.New("started_after must be earlier than started_before")
	}
	filter.StartedAfter = after
	filter.StartedBefore = before

	if query.Has("cursor") {
		cursor, err := records.ParseCursor(query.Get("cursor"))
		if err != nil {
			return records.Filter{}, errors.New("cursor is invalid")
		}
		filter.Cursor = &cursor
	}

	return filter, nil
}

func parseFilterTimestamp(query url.Values, name string) (*time.Time, error) {
	if !query.Has(name) {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, query.Get(name))
	if err != nil {
		return nil, errors.New(name + " must be an RFC3339 timestamp")
	}
	normalized := parsed.UTC()

	return &normalized, nil
}

func validFilterValue(value string, maximum int) bool {
	return len(value) > 0 && len(value) <= maximum && utf8.ValidString(value)
}

func knownRecordOutcome(value string) bool {
	switch value {
	case records.OutcomeCompleted, records.OutcomeFailed, records.OutcomeCanceled, records.OutcomeTimedOut:
		return true
	default:
		return false
	}
}
