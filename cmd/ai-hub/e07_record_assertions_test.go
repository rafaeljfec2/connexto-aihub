package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aihub/internal/execution"
	"aihub/internal/providers"
)

func TestE07RecordVerificationKeepsAssertionsIndependent(t *testing.T) {
	executionID := strings.Repeat("d", 32)
	requestID := "corr_independent_assertions"
	expectedRecord := e07ExecutionRecord{
		ExecutionID: executionID,
		RequestID:   &requestID,
		Provider:    providers.OpenCodeGoID,
		Outcome:     string(execution.OutcomeCompleted),
		Usage:       []byte("null"),
	}
	expectations := e07RecordExpectations{
		Records: map[string]e07ExpectedRecord{
			executionID: {RequestID: requestID, ExecutionID: executionID, Outcome: string(execution.OutcomeCompleted)},
		},
	}

	cases := []struct {
		name   string
		reader *e07FakeRecordReader
		check  func(*testing.T, e07RecordAssertions)
	}{
		{
			name: "list error does not suppress by-id",
			reader: &e07FakeRecordReader{
				listErr: errors.New("synthetic list failure"),
				records: map[string]e07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07RecordAssertions) {
				if assertions.ListOK {
					t.Fatal("list assertion accepted a failed list query")
				}
				if !assertions.ByIDOK || assertions.ByIDMatched != 1 {
					t.Fatal("by-id assertion was suppressed by the list failure")
				}
			},
		},
		{
			name: "outcome mismatch does not suppress by-id",
			reader: &e07FakeRecordReader{
				listRecords: []e07ExecutionRecord{{
					ExecutionID: executionID,
					RequestID:   &requestID,
					Provider:    providers.OpenCodeGoID,
					Outcome:     string(execution.OutcomeFailed),
					Usage:       []byte("null"),
				}},
				records: map[string]e07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07RecordAssertions) {
				if assertions.OutcomeOK {
					t.Fatal("outcome assertion accepted an E05 outcome mismatch")
				}
				if !assertions.ByIDOK {
					t.Fatal("by-id assertion was suppressed by the outcome mismatch")
				}
			},
		},
		{
			name: "missing record does not suppress by-id",
			reader: &e07FakeRecordReader{
				records: map[string]e07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07RecordAssertions) {
				if assertions.CorrelationOK || assertions.Matched != 0 {
					t.Fatal("correlation assertion accepted a missing record")
				}
				if !assertions.ByIDOK {
					t.Fatal("by-id assertion was suppressed by a missing list row")
				}
			},
		},
		{
			name: "schema mismatch does not suppress outcome or by-id",
			reader: &e07FakeRecordReader{
				listRecords: []e07ExecutionRecord{{
					ExecutionID: executionID,
					RequestID:   &requestID,
					Provider:    providers.OpenCodeGoID,
					Outcome:     string(execution.OutcomeCompleted),
					Usage:       []byte(`{"total_tokens":1}`),
				}},
				records: map[string]e07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07RecordAssertions) {
				if assertions.SchemaOK {
					t.Fatal("schema assertion accepted a non-null usage value")
				}
				if !assertions.OutcomeOK || !assertions.ByIDOK {
					t.Fatal("schema mismatch suppressed another assertion")
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assertions := e07AssertRecords(context.Background(), testCase.reader, expectations)
			testCase.check(t, assertions)
		})
	}
}

func TestE07RecordAssertionsSurviveParentCancellation(t *testing.T) {
	executionID := strings.Repeat("f", 32)
	requestID := "corr_parent_cancellation"
	expectedRecord := e07ExecutionRecord{
		ExecutionID: executionID,
		RequestID:   &requestID,
		Provider:    providers.OpenCodeGoID,
		Outcome:     string(execution.OutcomeCompleted),
		Usage:       []byte("null"),
	}
	parent, cancel := context.WithCancel(context.Background())
	reader := &e07FakeRecordReader{
		listErr: errors.New("synthetic slow list failure"),
		records: map[string]e07ExecutionRecord{executionID: expectedRecord},
		onList:  cancel,
	}
	expectations := e07RecordExpectations{
		Records: map[string]e07ExpectedRecord{
			executionID: {RequestID: requestID, ExecutionID: executionID, Outcome: string(execution.OutcomeCompleted)},
		},
	}

	assertions := e07AssertRecords(parent, reader, expectations)
	if !assertions.ByIDOK {
		t.Fatalf("by-id assertion was suppressed by parent cancellation: %s", assertions.Summary("cancelled_parent"))
	}
	if len(reader.getContextErrs) == 0 {
		t.Fatal("by-id query was not attempted")
	}
	for index, err := range reader.getContextErrs {
		if err != nil {
			t.Fatalf("by-id query %d inherited the cancelled parent: %v", index, err)
		}
	}
}

type e07FakeRecordReader struct {
	listRecords    []e07ExecutionRecord
	listErr        error
	records        map[string]e07ExecutionRecord
	getErr         error
	onList         func()
	contextErrs    []error
	getContextErrs []error
}

func (reader *e07FakeRecordReader) List(ctx context.Context) ([]e07ExecutionRecord, error) {
	reader.contextErrs = append(reader.contextErrs, ctx.Err())
	if reader.onList != nil {
		reader.onList()
	}
	if reader.listErr != nil {
		return nil, reader.listErr
	}
	return append([]e07ExecutionRecord(nil), reader.listRecords...), nil
}

func (reader *e07FakeRecordReader) Get(ctx context.Context, executionID string) (e07ExecutionRecord, error) {
	reader.getContextErrs = append(reader.getContextErrs, ctx.Err())
	if reader.getErr != nil {
		return e07ExecutionRecord{}, reader.getErr
	}
	record, ok := reader.records[executionID]
	if !ok {
		return e07ExecutionRecord{}, errors.New("record not found")
	}
	return record, nil
}
