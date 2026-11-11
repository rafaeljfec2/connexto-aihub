package records

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"
	"unicode/utf8"
)

type recordWire struct {
	SchemaVersion     *int            `json:"schema_version"`
	ExecutionID       *string         `json:"execution_id"`
	RequestID         *string         `json:"request_id"`
	ProviderID        *string         `json:"provider_id"`
	ModelID           *string         `json:"model_id"`
	StartedAt         *string         `json:"started_at"`
	CompletedAt       *string         `json:"completed_at"`
	DurationNS        *int64          `json:"duration_ns"`
	Outcome           *string         `json:"outcome"`
	Phase             *string         `json:"phase"`
	Cause             *string         `json:"cause"`
	UpstreamStatus    *int            `json:"upstream_status"`
	ResponseCommitted *bool           `json:"response_committed"`
	Usage             json.RawMessage `json:"usage"`
}

// encodeRecord validates and serializes one record as a single JSON line,
// including its trailing LF.
func encodeRecord(record Record) ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}

	wire := recordWire{
		SchemaVersion:     pointer(record.SchemaVersion),
		ExecutionID:       pointer(record.ExecutionID),
		RequestID:         record.RequestID,
		ProviderID:        pointer(record.ProviderID),
		ModelID:           record.ModelID,
		StartedAt:         pointer(formatTimestamp(record.StartedAt)),
		CompletedAt:       pointer(formatTimestamp(record.CompletedAt)),
		DurationNS:        pointer(record.DurationNS),
		Outcome:           pointer(record.Outcome),
		Phase:             pointer(record.Phase),
		Cause:             record.Cause,
		UpstreamStatus:    record.UpstreamStatus,
		ResponseCommitted: pointer(record.ResponseCommitted),
		Usage:             json.RawMessage("null"),
	}

	encoded, err := json.Marshal(wire)
	if err != nil {
		return nil, invalidRecord("encode record: %v", err)
	}

	if len(encoded)+1 > maxRecordLineBytes {
		return nil, invalidRecord("encoded record exceeds %d bytes", maxRecordLineBytes)
	}

	return append(encoded, '\n'), nil
}

// decodeRecord parses one JSON object without its framing LF.
func decodeRecord(line []byte) (Record, error) {
	if len(line) == 0 || len(line)+1 > maxRecordLineBytes {
		return Record{}, invalidRecord("record line length is outside its bounds")
	}
	if !utf8.Valid(line) || bytes.ContainsRune(line, '\n') {
		return Record{}, invalidRecord("record line is not single-line UTF-8")
	}

	var wire recordWire
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return Record{}, invalidRecord("decode record: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Record{}, invalidRecord("record line carries trailing data")
	}
	if err := requireExactFields(line); err != nil {
		return Record{}, err
	}

	record, err := wire.record()
	if err != nil {
		return Record{}, err
	}
	if err := record.Validate(); err != nil {
		return Record{}, err
	}

	return record, nil
}

func (w recordWire) record() (Record, error) {
	if w.SchemaVersion == nil || w.ExecutionID == nil || w.ProviderID == nil ||
		w.StartedAt == nil || w.CompletedAt == nil || w.DurationNS == nil ||
		w.Outcome == nil || w.Phase == nil || w.ResponseCommitted == nil || w.Usage == nil {
		return Record{}, invalidRecord("required field is missing")
	}
	if !bytes.Equal(w.Usage, []byte("null")) {
		return Record{}, invalidRecord("schema v1 requires usage to be null")
	}

	startedAt, err := parseTimestamp(*w.StartedAt)
	if err != nil {
		return Record{}, err
	}
	completedAt, err := parseTimestamp(*w.CompletedAt)
	if err != nil {
		return Record{}, err
	}

	return Record{
		SchemaVersion:     *w.SchemaVersion,
		ExecutionID:       *w.ExecutionID,
		RequestID:         w.RequestID,
		ProviderID:        *w.ProviderID,
		ModelID:           w.ModelID,
		StartedAt:         startedAt,
		CompletedAt:       completedAt,
		DurationNS:        *w.DurationNS,
		Outcome:           *w.Outcome,
		Phase:             *w.Phase,
		Cause:             w.Cause,
		UpstreamStatus:    w.UpstreamStatus,
		ResponseCommitted: *w.ResponseCommitted,
	}, nil
}

var recordFieldNames = []string{
	"schema_version", "execution_id", "request_id", "provider_id", "model_id",
	"started_at", "completed_at", "duration_ns", "outcome", "phase", "cause",
	"upstream_status", "response_committed", "usage",
}

func requireExactFields(line []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		return invalidRecord("decode record fields: %v", err)
	}
	if len(fields) != len(recordFieldNames) {
		return invalidRecord("record must contain exactly the schema-v1 fields")
	}

	for _, name := range recordFieldNames {
		if _, ok := fields[name]; !ok {
			return invalidRecord("required field %q is missing", name)
		}
	}

	return nil
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, invalidRecord("invalid timestamp %q", value)
	}
	if formatTimestamp(parsed) != value {
		return time.Time{}, invalidRecord("timestamp is not canonical UTC RFC3339Nano")
	}

	return parsed.UTC(), nil
}

func pointer[T any](value T) *T {
	return &value
}
