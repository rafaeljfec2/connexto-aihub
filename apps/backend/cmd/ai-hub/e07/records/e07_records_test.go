package records

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"aihub/internal/execution"
	"aihub/internal/providers"
	"net/http"
	"strings"
	"testing"
)

func TestE07ExpectedRecordsDeriveOutcomeFromE05TerminalObservation(t *testing.T) {
	captures := []e07support.E07RequestCapture{
		{ID: "corr_request_400", Status: http.StatusBadRequest},
		{ID: "corr_request_200", Status: http.StatusOK},
		{ID: "corr_request_local_failure", Status: http.StatusBadGateway},
		{ID: "corr_request_upstream_401", Status: http.StatusUnauthorized},
		{ID: "corr_authentication_probe", Status: http.StatusUnauthorized},
		{ID: "corr_request_unobserved", Status: http.StatusOK},
	}
	terminals := []e07support.E07TerminalObservation{
		{RequestID: "corr_request_400", ExecutionID: strings.Repeat("a", 32), ProviderID: providers.OpenCodeGoID, Outcome: string(execution.OutcomeCompleted)},
		{RequestID: "corr_request_200", ExecutionID: strings.Repeat("b", 32), ProviderID: providers.OpenCodeGoID, Outcome: string(execution.OutcomeCompleted)},
		{RequestID: "corr_request_local_failure", ExecutionID: strings.Repeat("c", 32), ProviderID: providers.OpenCodeGoID, Outcome: string(execution.OutcomeFailed)},
		{RequestID: "corr_request_upstream_401", ExecutionID: strings.Repeat("d", 32), ProviderID: providers.OpenCodeGoID, Outcome: string(execution.OutcomeCompleted)},
	}

	expectations := e07support.E07BuildExpectedRecords(captures, terminals, []string{"corr_authentication_probe"})

	if got, want := len(expectations.Records), 4; got != want {
		t.Fatalf("expected records = %d, want %d", got, want)
	}
	if got, want := expectations.UnmatchedCaptures, 1; got != want {
		t.Fatalf("unmatched captures = %d, want %d", got, want)
	}
	if got := expectations.Records[strings.Repeat("a", 32)].Outcome; got != string(execution.OutcomeCompleted) {
		t.Fatalf("upstream 4xx expectation = %q, want the E05 completed outcome", got)
	}
	if got := expectations.Records[strings.Repeat("c", 32)].Outcome; got != string(execution.OutcomeFailed) {
		t.Fatalf("local failure expectation = %q, want the E05 failed outcome", got)
	}
	if got := expectations.Records[strings.Repeat("d", 32)].Outcome; got != string(execution.OutcomeCompleted) {
		t.Fatalf("upstream 401 expectation = %q, want the E05 completed outcome", got)
	}
	for _, record := range expectations.Records {
		if record.RequestID == "corr_authentication_probe" {
			t.Fatal("a request rejected before E05 admission produced an expectation")
		}
		if record.RequestID == "corr_request_unobserved" {
			t.Fatal("a capture without an E05 terminal observation produced an expectation")
		}
	}
}
