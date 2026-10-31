package records

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRecordValidateAcceptsSchemaV1Records(t *testing.T) {
	completed := completedRecord(executionID(1), at(10))
	if err := completed.Validate(); err != nil {
		t.Fatalf("completed record rejected: %v", err)
	}

	failed := failedRecord(executionID(2), at(11), CauseTransport)
	if err := failed.Validate(); err != nil {
		t.Fatalf("failed record rejected: %v", err)
	}

	nullable := failedRecord(executionID(3), at(12), CauseInactivity)
	nullable.RequestID = nil
	nullable.ModelID = nil
	if err := nullable.Validate(); err != nil {
		t.Fatalf("record with null optional fields rejected: %v", err)
	}
}

func TestRecordValidateRejectsInvalidValues(t *testing.T) {
	model := strings.Repeat("m", 4096)

	tests := []struct {
		name   string
		mutate func(*Record)
	}{
		{name: "schema version", mutate: func(r *Record) { r.SchemaVersion = 2 }},
		{name: "execution id length", mutate: func(r *Record) { r.ExecutionID = "abc" }},
		{name: "execution id uppercase", mutate: func(r *Record) { r.ExecutionID = strings.ToUpper(executionID(0xabcdef)) }},
		{name: "execution id non hex", mutate: func(r *Record) { r.ExecutionID = strings.Repeat("z", 32) }},
		{name: "request id short", mutate: func(r *Record) { r.RequestID = strPtr("short") }},
		{name: "request id invalid character", mutate: func(r *Record) { r.RequestID = strPtr("req with spaces") }},
		{name: "provider empty", mutate: func(r *Record) { r.ProviderID = "" }},
		{name: "provider oversize", mutate: func(r *Record) { r.ProviderID = strings.Repeat("p", 129) }},
		{name: "model empty", mutate: func(r *Record) { r.ModelID = strPtr("") }},
		{name: "model oversize", mutate: func(r *Record) { r.ModelID = strPtr(model + "m") }},
		{name: "model invalid utf8", mutate: func(r *Record) { r.ModelID = strPtr("\xff") }},
		{name: "started zero", mutate: func(r *Record) { r.StartedAt = time.Time{} }},
		{name: "started not utc", mutate: func(r *Record) { r.StartedAt = r.StartedAt.In(time.FixedZone("x", 3600)) }},
		{name: "completed zero", mutate: func(r *Record) { r.CompletedAt = time.Time{} }},
		{name: "duration negative", mutate: func(r *Record) { r.DurationNS = -1 }},
		{name: "completed mismatch", mutate: func(r *Record) { r.CompletedAt = r.CompletedAt.Add(time.Nanosecond) }},
		{name: "outcome unknown", mutate: func(r *Record) { r.Outcome = "unknown" }},
		{name: "phase unknown", mutate: func(r *Record) { r.Phase = "unknown" }},
		{name: "cause on completed", mutate: func(r *Record) { r.Cause = strPtr(CauseTransport) }},
		{name: "missing cause on failure", mutate: func(r *Record) { r.Outcome = OutcomeFailed }},
		{name: "unknown cause", mutate: func(r *Record) { r.Outcome = OutcomeFailed; r.Cause = strPtr("because") }},
		{name: "status below 100", mutate: func(r *Record) { r.UpstreamStatus = intPtr(99) }},
		{name: "status above 599", mutate: func(r *Record) { r.UpstreamStatus = intPtr(600) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := completedRecord(executionID(1), at(10))
			test.mutate(&record)

			if err := record.Validate(); !errors.Is(err, ErrInvalidRecord) {
				t.Fatalf("Validate() error = %v, want ErrInvalidRecord", err)
			}
		})
	}
}

func TestEncodeRecordRoundTripsEveryField(t *testing.T) {
	record := failedRecord(executionID(7), at(9), CauseUpstreamRead)
	record.ModelID = strPtr("vendor/model")
	record.UpstreamStatus = intPtr(502)

	line, err := encodeRecord(record)
	if err != nil {
		t.Fatalf("encodeRecord() returned error: %v", err)
	}

	if !strings.HasSuffix(string(line), "\n") || strings.Count(string(line), "\n") != 1 {
		t.Fatalf("encoded line must end with exactly one LF: %q", line)
	}

	decoded, err := decodeRecord(line[:len(line)-1])
	if err != nil {
		t.Fatalf("decodeRecord() returned error: %v", err)
	}

	if decoded.ExecutionID != record.ExecutionID ||
		decoded.Outcome != record.Outcome ||
		decoded.Phase != record.Phase ||
		decoded.DurationNS != record.DurationNS ||
		decoded.ResponseCommitted != record.ResponseCommitted ||
		!decoded.StartedAt.Equal(record.StartedAt) ||
		!decoded.CompletedAt.Equal(record.CompletedAt) ||
		decoded.Cause == nil || *decoded.Cause != CauseUpstreamRead ||
		decoded.ModelID == nil || *decoded.ModelID != "vendor/model" ||
		decoded.UpstreamStatus == nil || *decoded.UpstreamStatus != 502 {
		t.Fatalf("decoded record = %+v, want %+v", decoded, record)
	}
}

func TestEncodeRecordEmitsUsageNullAndNullFields(t *testing.T) {
	record := completedRecord(executionID(4), at(4))
	record.RequestID = nil
	record.ModelID = nil
	record.UpstreamStatus = nil

	line, err := encodeRecord(record)
	if err != nil {
		t.Fatalf("encodeRecord() returned error: %v", err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(line[:len(line)-1], &payload); err != nil {
		t.Fatalf("encoded line is not JSON: %v", err)
	}

	for _, field := range []string{"request_id", "model_id", "cause", "upstream_status"} {
		if string(payload[field]) != "null" {
			t.Fatalf("%s = %s, want null", field, payload[field])
		}
	}

	if string(payload["usage"]) != "null" {
		t.Fatalf("usage = %s, want null", payload["usage"])
	}
}

func TestEncodeRecordRejectsOversizedLine(t *testing.T) {
	record := completedRecord(executionID(5), at(5))
	record.ModelID = strPtr(strings.Repeat("\x00", 4096))

	if _, err := encodeRecord(record); !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("encodeRecord() error = %v, want ErrInvalidRecord", err)
	}
}

func TestDecodeRecordRejectsMalformedLines(t *testing.T) {
	valid := completedRecord(executionID(6), at(6))
	line, err := encodeRecord(valid)
	if err != nil {
		t.Fatalf("encodeRecord() returned error: %v", err)
	}
	body := string(line[:len(line)-1])

	tests := []struct {
		name string
		line string
	}{
		{name: "invalid json", line: `{"schema_version":`},
		{name: "unknown field", line: strings.Replace(body, `"usage":null`, `"usage":null,"extra":1`, 1)},
		{name: "missing usage", line: strings.Replace(body, `,"usage":null`, "", 1)},
		{name: "usage object", line: strings.Replace(body, `"usage":null`, `"usage":{}`, 1)},
		{name: "null execution id", line: strings.Replace(body, `"execution_id":"`+executionID(6)+`"`, `"execution_id":null`, 1)},
		{name: "trailing data", line: body + `{}`},
		{name: "invalid utf8", line: strings.Replace(body, `"opencode-go"`, "\"\xff\"", 1)},
		{name: "non canonical timestamp", line: strings.Replace(body, `2026-10-31T12:00:06.123456789Z`, `2026-10-31T12:00:06.123456789+00:00`, 1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeRecord([]byte(test.line)); !errors.Is(err, ErrInvalidRecord) {
				t.Fatalf("decodeRecord() error = %v, want ErrInvalidRecord", err)
			}
		})
	}
}

func TestDecodeRecordAcceptsNullableFields(t *testing.T) {
	record := completedRecord(executionID(8), at(8))
	record.RequestID = nil
	record.ModelID = nil
	record.UpstreamStatus = nil

	line, err := encodeRecord(record)
	if err != nil {
		t.Fatalf("encodeRecord() returned error: %v", err)
	}

	decoded, err := decodeRecord(line[:len(line)-1])
	if err != nil {
		t.Fatalf("decodeRecord() returned error: %v", err)
	}

	if decoded.RequestID != nil || decoded.ModelID != nil || decoded.UpstreamStatus != nil {
		t.Fatalf("decoded nullable fields = %+v, want nil", decoded)
	}
}
