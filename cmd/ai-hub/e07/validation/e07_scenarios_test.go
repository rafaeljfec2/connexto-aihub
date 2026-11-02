package validation

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"strings"
	"testing"
)

func TestE07ReplayDiagnosticsSeparateEveryCondition(t *testing.T) {
	first, second := e07support.E07ReplayFixture()
	healthy := e07support.E07ReplayObservation{PriorContextPresent: true, PreviousResponseIDPresent: false}

	cases := []struct {
		name        string
		first       e07support.E07CodexRun
		second      e07support.E07CodexRun
		requests    int
		observation e07support.E07ReplayObservation
		wantResult  string
		wantFields  []string
	}{
		{
			name: "all conditions satisfied", first: first, second: second, requests: 2,
			observation: healthy, wantResult: "PASS", wantFields: nil,
		},
		{
			name: "first turn failed", first: e07support.E07CodexRun{Ran: true, ExitCode: 1}, second: e07support.E07CodexRun{},
			requests: 1, observation: e07support.E07ReplayObservation{}, wantResult: "FAIL",
			wantFields: []string{"first_turn=FAIL", "second_turn=FAIL"},
		},
		{
			name: "second turn failed", first: first, second: e07support.E07CodexRun{Ran: true, ExitCode: 1},
			requests: 2, observation: healthy, wantResult: "FAIL",
			wantFields: []string{"first_turn=PASS", "second_turn=FAIL"},
		},
		{
			name: "unexpected request count", first: first, second: second, requests: 1,
			observation: healthy, wantResult: "FAIL",
			wantFields: []string{"requests=1", "request_count=FAIL"},
		},
		{
			name: "marker not recalled", first: first, second: e07support.E07CodexRun{Ran: true, ExitCode: 0, Message: "no marker"},
			requests: 2, observation: healthy, wantResult: "FAIL",
			wantFields: []string{"marker_recall=FAIL"},
		},
		{
			name: "prior context missing", first: first, second: second, requests: 2,
			observation: e07support.E07ReplayObservation{PriorContextPresent: false, PreviousResponseIDPresent: false},
			wantResult:  "FAIL", wantFields: []string{"prior_context=FAIL"},
		},
		{
			name: "previous_response_id present", first: first, second: second, requests: 2,
			observation: e07support.E07ReplayObservation{PriorContextPresent: true, PreviousResponseIDPresent: true},
			wantResult:  "FAIL", wantFields: []string{"previous_response_id_absent=FAIL"},
		},
		{
			name: "environment limitation", first: e07support.E07CodexRun{Ran: true, ExitCode: 1, EnvironmentLimit: "sandbox_or_approval"},
			second: e07support.E07CodexRun{}, requests: 0, observation: e07support.E07ReplayObservation{},
			wantResult: "NOT VERIFIED", wantFields: []string{"environment_limited=true"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			diagnostics := e07support.E07EvaluateReplay(testCase.first, testCase.second, testCase.requests,
				e07support.E07ReplayExpectedRequests, testCase.observation, e07support.E07ReplayMarker)
			if got := diagnostics.Result(); got != testCase.wantResult {
				t.Fatalf("replay result = %q, want %q", got, testCase.wantResult)
			}
			summary := diagnostics.Summary()
			for _, field := range testCase.wantFields {
				if !strings.Contains(summary, field) {
					t.Fatalf("replay summary %q is missing %q", summary, field)
				}
			}
			for _, forbidden := range []string{e07support.E07ReplayMarker, "thr_synthetic_identifier", "previous turn"} {
				if strings.Contains(summary, forbidden) {
					t.Fatalf("replay summary exposes content %q", forbidden)
				}
			}
		})
	}
}

func TestE07ReplayDiagnosticsRequireBothTurnsToReportSuccess(t *testing.T) {
	first, second := e07support.E07ReplayFixture()
	diagnostics := e07support.E07EvaluateReplay(first, second, e07support.E07ReplayExpectedRequests,
		e07support.E07ReplayExpectedRequests, e07support.E07ReplayObservation{PriorContextPresent: true}, e07support.E07ReplayMarker)

	if !diagnostics.FirstTurnOK || !diagnostics.SecondTurnOK || !diagnostics.MarkerRecallOK ||
		!diagnostics.PriorContextOK || !diagnostics.PreviousResponseIDAbsent || !diagnostics.RequestCountOK {
		t.Fatalf("healthy replay diagnostics are incomplete: %s", diagnostics.Summary())
	}
}

func TestE07ScenarioResultRequiresPassExceptCodexProgressiveRendering(t *testing.T) {
	cases := []struct {
		name       string
		result     string
		shouldFail bool
	}{
		{name: "basic_generation", result: "PASS"},
		{name: "basic_generation", result: "NOT VERIFIED", shouldFail: true},
		{name: "tool_round_trip", result: "NOT RUN — environment/provider limitation", shouldFail: true},
		{name: "codex_progressive_rendering", result: "NOT VERIFIED"},
		{name: "codex_progressive_rendering", result: "FAIL", shouldFail: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name+"/"+testCase.result, func(t *testing.T) {
			if got := e07support.E07ScenarioResultMustFail(testCase.name, testCase.result); got != testCase.shouldFail {
				t.Fatalf("scenario result failure = %t, want %t", got, testCase.shouldFail)
			}
		})
	}
}
