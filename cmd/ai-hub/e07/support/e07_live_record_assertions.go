package support

import (
	"bytes"
	"context"
	"strconv"
	"time"

	"aihub/internal/providers"
)

// e07BuildExpectedRecords joins captured gateway requests with the E05 terminal
// observations recorded by the validation observer. The gateway status is
// irrelevant to the expected outcome; a fully read upstream 4xx response is a
// completed E05 execution, while a transport or body-read failure is not.
// Requests rejected before E05 admission are excluded by identifier (the
// authentication probe), never by status.
func E07BuildExpectedRecords(captures []E07RequestCapture, terminals []E07TerminalObservation, preExecution []string) E07RecordExpectations {
	byRequest := make(map[string]E07TerminalObservation, len(terminals))
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

	expectations := E07RecordExpectations{Records: make(map[string]E07ExpectedRecord, len(terminals))}
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
		expectations.Records[terminal.ExecutionID] = E07ExpectedRecord{
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
func E07AssertRecords(ctx context.Context, reader E07RecordReader, expectations E07RecordExpectations) E07RecordAssertions {
	return E07AssertRecordsWithin(ctx, reader, expectations, time.Time{})
}

// e07AssertRecordsWithin bounds every query by the remaining poll window as well
// as the per-query timeout.
func E07AssertRecordsWithin(ctx context.Context, reader E07RecordReader, expectations E07RecordExpectations, deadline time.Time) E07RecordAssertions {
	assertions := E07RecordAssertions{Expected: expectations.Len(), Unmatched: expectations.UnmatchedCaptures}

	listContext, cancelList := E07RecordQueryContext(ctx, deadline)
	records, listErr := reader.List(listContext)
	cancelList()
	if listErr == nil {
		assertions.ListOK = true
		assertions.CorrelationOK = expectations.Len() > 0
		assertions.OutcomeOK = expectations.Len() > 0
		assertions.SchemaOK = expectations.Len() > 0

		listed := make(map[string]E07ExecutionRecord, len(records))
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
			if !E07RecordSchemaValid(record) {
				assertions.SchemaOK = false
			}
		}
	}

	assertions.ByIDOK = expectations.Len() > 0
	for executionID, expected := range expectations.Records {
		assertions.ByIDRequested++
		getContext, cancelGet := E07RecordQueryContext(ctx, deadline)
		record, err := reader.Get(getContext, executionID)
		cancelGet()
		if err != nil || !E07RecordMatchesExpected(record, expected) {
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
func E07RecordQueryContext(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	timeout := E07RecordQueryTimeout
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

func E07RecordMatchesExpected(record E07ExecutionRecord, expected E07ExpectedRecord) bool {
	return record.ExecutionID == expected.ExecutionID && record.RequestID != nil &&
		*record.RequestID == expected.RequestID && record.Outcome == expected.Outcome &&
		E07RecordSchemaValid(record)
}

func E07RecordSchemaValid(record E07ExecutionRecord) bool {
	return record.Provider == providers.OpenCodeGoID && record.ModelID == nil &&
		bytes.Equal(record.Usage, []byte("null"))
}

// e07AssertRecordsEventually polls within the caller's window so a record that
// is still being written does not produce an immediate false failure.
func E07AssertRecordsEventually(ctx context.Context, reader E07RecordReader, expectations E07RecordExpectations, window time.Duration) E07RecordAssertions {
	deadline := time.Now().Add(window)
	for {
		assertions := E07AssertRecordsWithin(ctx, reader, expectations, deadline)
		if assertions.Pass() || ctx.Err() != nil || time.Now().After(deadline) {
			return assertions
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// e07AssertRecordsAfterRestart always derives a fresh context from the parent's
// values while discarding its deadline and cancellation, so an expired
// pre-stop context cannot suppress the post-restart verification.
func E07AssertRecordsAfterRestart(parent context.Context, reader E07RecordReader, expectations E07RecordExpectations, window time.Duration) E07RecordAssertions {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), window+5*time.Second)
	defer cancel()

	deadline := time.Now().Add(window)
	for {
		assertions := E07AssertRecordsWithin(ctx, reader, expectations, deadline)
		if assertions.Pass() || time.Now().After(deadline) {
			return assertions
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (assertions E07RecordAssertions) Summary(phase string) string {
	return "E07 records phase=" + phase +
		" list=" + E07IntegrityResult(assertions.ListOK) +
		" correlation=" + E07IntegrityResult(assertions.CorrelationOK) +
		" outcome=" + E07IntegrityResult(assertions.OutcomeOK) +
		" schema=" + E07IntegrityResult(assertions.SchemaOK) +
		" by_id=" + E07IntegrityResult(assertions.ByIDOK) +
		" expected=" + strconv.Itoa(assertions.Expected) +
		" matched=" + strconv.Itoa(assertions.Matched) +
		" by_id_requested=" + strconv.Itoa(assertions.ByIDRequested) +
		" by_id_matched=" + strconv.Itoa(assertions.ByIDMatched) +
		" unmatched_captures=" + strconv.Itoa(assertions.Unmatched)
}
