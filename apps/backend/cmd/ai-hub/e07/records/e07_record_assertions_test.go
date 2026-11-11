package records

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"aihub/internal/execution"
	"aihub/internal/providers"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestE07RecordVerificationKeepsAssertionsIndependent(t *testing.T) {
	executionID := strings.Repeat("d", 32)
	requestID := "corr_independent_assertions"
	expectedRecord := e07support.E07ExecutionRecord{
		ExecutionID: executionID,
		RequestID:   &requestID,
		Provider:    providers.OpenCodeGoID,
		Outcome:     string(execution.OutcomeCompleted),
		Usage:       []byte("null"),
	}
	expectations := e07support.E07RecordExpectations{
		Records: map[string]e07support.E07ExpectedRecord{
			executionID: {RequestID: requestID, ExecutionID: executionID, Outcome: string(execution.OutcomeCompleted)},
		},
	}

	cases := []struct {
		name   string
		Reader *e07support.E07FakeRecordReader
		check  func(*testing.T, e07support.E07RecordAssertions)
	}{
		{
			name: "list error does not suppress by-id",
			Reader: &e07support.E07FakeRecordReader{
				ListErr: errors.New("synthetic list failure"),
				Records: map[string]e07support.E07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07support.E07RecordAssertions) {
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
			Reader: &e07support.E07FakeRecordReader{
				ListRecords: []e07support.E07ExecutionRecord{{
					ExecutionID: executionID,
					RequestID:   &requestID,
					Provider:    providers.OpenCodeGoID,
					Outcome:     string(execution.OutcomeFailed),
					Usage:       []byte("null"),
				}},
				Records: map[string]e07support.E07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07support.E07RecordAssertions) {
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
			Reader: &e07support.E07FakeRecordReader{
				Records: map[string]e07support.E07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07support.E07RecordAssertions) {
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
			Reader: &e07support.E07FakeRecordReader{
				ListRecords: []e07support.E07ExecutionRecord{{
					ExecutionID: executionID,
					RequestID:   &requestID,
					Provider:    providers.OpenCodeGoID,
					Outcome:     string(execution.OutcomeCompleted),
					Usage:       []byte(`{"total_tokens":1}`),
				}},
				Records: map[string]e07support.E07ExecutionRecord{executionID: expectedRecord},
			},
			check: func(t *testing.T, assertions e07support.E07RecordAssertions) {
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
			assertions := e07support.E07AssertRecords(context.Background(), testCase.Reader, expectations)
			testCase.check(t, assertions)
		})
	}
}

func TestE07RecordAssertionsSurviveParentCancellation(t *testing.T) {
	executionID := strings.Repeat("f", 32)
	requestID := "corr_parent_cancellation"
	expectedRecord := e07support.E07ExecutionRecord{
		ExecutionID: executionID,
		RequestID:   &requestID,
		Provider:    providers.OpenCodeGoID,
		Outcome:     string(execution.OutcomeCompleted),
		Usage:       []byte("null"),
	}
	parent, cancel := context.WithCancel(context.Background())
	reader := &e07support.E07FakeRecordReader{
		ListErr: errors.New("synthetic slow list failure"),
		Records: map[string]e07support.E07ExecutionRecord{executionID: expectedRecord},
		OnList:  cancel,
	}
	expectations := e07support.E07RecordExpectations{
		Records: map[string]e07support.E07ExpectedRecord{
			executionID: {RequestID: requestID, ExecutionID: executionID, Outcome: string(execution.OutcomeCompleted)},
		},
	}

	assertions := e07support.E07AssertRecords(parent, reader, expectations)
	if !assertions.ByIDOK {
		t.Fatalf("by-id assertion was suppressed by parent cancellation: %s", assertions.Summary("cancelled_parent"))
	}
	if len(reader.GetContextErrs) == 0 {
		t.Fatal("by-id query was not attempted")
	}
	for index, err := range reader.GetContextErrs {
		if err != nil {
			t.Fatalf("by-id query %d inherited the cancelled parent: %v", index, err)
		}
	}
}
