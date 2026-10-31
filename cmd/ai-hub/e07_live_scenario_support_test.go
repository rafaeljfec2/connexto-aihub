package main

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aihub/internal/providers"
)

const (
	e07ToolMarker             = "E07-TOOL-MARKER-9A31"
	e07ReplayExpectedRequests = 2
)

// e07UpstreamProbeObservation keeps status and body observations separate and
// never retains body content.
type e07UpstreamProbeObservation struct {
	DirectStatus    int
	HubStatus       int
	EnvelopePresent bool
	HubBodyRead     bool
}

// PassthroughObserved reports the real-provider scenario invariant: the Hub
// returned the upstream client-error class with a generic error envelope. Body
// equality across independent provider calls is deliberately not asserted.
func (observation e07UpstreamProbeObservation) PassthroughObserved() bool {
	return observation.HubBodyRead && observation.DirectStatus >= http.StatusBadRequest &&
		observation.DirectStatus < http.StatusInternalServerError &&
		observation.HubStatus == observation.DirectStatus && observation.EnvelopePresent
}

func (observation e07UpstreamProbeObservation) Summary() string {
	return "E07 upstream_error direct_status=" + strconv.Itoa(observation.DirectStatus) +
		" hub_status=" + strconv.Itoa(observation.HubStatus) +
		" status_preserved=" + strconv.FormatBool(observation.HubStatus == observation.DirectStatus) +
		" envelope=" + strconv.FormatBool(observation.EnvelopePresent) +
		" body_read=" + strconv.FormatBool(observation.HubBodyRead)
}

// e07ReplayDiagnostics separates every independent replay condition so a single
// aggregate FAIL can never hide which predicate failed.
type e07ReplayDiagnostics struct {
	FirstTurnOK              bool
	SecondTurnOK             bool
	RequestCountOK           bool
	MarkerRecallOK           bool
	PriorContextOK           bool
	PreviousResponseIDAbsent bool
	Requests                 int
	ExpectedRequests         int
	EnvironmentLimited       bool
}

func e07EvaluateReplay(first, second e07CodexRun, requests, expectedRequests int, observation e07ReplayObservation, marker string) e07ReplayDiagnostics {
	diagnostics := e07ReplayDiagnostics{
		FirstTurnOK:              first.Ran && first.ExitCode == 0 && first.ThreadID != "",
		SecondTurnOK:             second.Ran && second.ExitCode == 0,
		RequestCountOK:           expectedRequests > 0 && requests == expectedRequests,
		MarkerRecallOK:           marker != "" && strings.Contains(second.Message, marker),
		PriorContextOK:           observation.PriorContextPresent,
		PreviousResponseIDAbsent: !observation.PreviousResponseIDPresent,
		Requests:                 requests,
		ExpectedRequests:         expectedRequests,
		EnvironmentLimited:       first.EnvironmentLimit != "" || second.EnvironmentLimit != "",
	}

	return diagnostics
}

func (diagnostics e07ReplayDiagnostics) satisfied() bool {
	return diagnostics.FirstTurnOK && diagnostics.SecondTurnOK && diagnostics.RequestCountOK &&
		diagnostics.MarkerRecallOK && diagnostics.PriorContextOK && diagnostics.PreviousResponseIDAbsent
}

func (diagnostics e07ReplayDiagnostics) Result() string {
	if diagnostics.satisfied() {
		return "PASS"
	}
	if diagnostics.EnvironmentLimited {
		return "NOT VERIFIED"
	}

	return "FAIL"
}

// Summary emits structural booleans and aggregate counts only. It never
// contains prompts, responses, identifiers, or payload bytes.
func (diagnostics e07ReplayDiagnostics) Summary() string {
	return "E07 replay first_turn=" + e07IntegrityResult(diagnostics.FirstTurnOK) +
		" second_turn=" + e07IntegrityResult(diagnostics.SecondTurnOK) +
		" requests=" + strconv.Itoa(diagnostics.Requests) +
		" expected_requests=" + strconv.Itoa(diagnostics.ExpectedRequests) +
		" request_count=" + e07IntegrityResult(diagnostics.RequestCountOK) +
		" marker_recall=" + e07IntegrityResult(diagnostics.MarkerRecallOK) +
		" prior_context=" + e07IntegrityResult(diagnostics.PriorContextOK) +
		" previous_response_id_absent=" + e07IntegrityResult(diagnostics.PreviousResponseIDAbsent) +
		" environment_limited=" + strconv.FormatBool(diagnostics.EnvironmentLimited)
}

func e07RunCodex(setup *e07LiveSetup, workspace, prompt string, args ...string) e07CodexRun {
	result, err := runE07Codex(setup.CodexPath, setup.CodexHome, setup.GatewayKey, workspace, prompt, args...)
	if err != nil && result.EnvironmentLimit == "" {
		result.ExitCode = 1
	}
	return result
}

func e07CodexResult(result e07CodexRun, expected string) string {
	if result.Ran && result.ExitCode == 0 && strings.Contains(result.Message, expected) {
		return "PASS"
	}
	return "FAIL"
}

func e07ProgressiveResult(result e07CodexRun) string {
	if result.ProgressiveSeen {
		return "PASS"
	}
	return "NOT VERIFIED"
}

func e07ScenarioOutcome(passed bool) string {
	if passed {
		return "PASS"
	}

	return "FAIL"
}

func e07ScenarioResultMustFail(name, result string) bool {
	if result == "PASS" {
		return false
	}
	return name != "codex_progressive_rendering" || result != "NOT VERIFIED"
}

func e07CodexRequestsReachHub(startRequests int, startCalls uint64, hub *e07Hub, observer *e07ProviderObserver) (bool, int) {
	captures := hub.requestIDs.snapshot()
	if startRequests > len(captures) {
		return false, 0
	}
	count := len(captures) - startRequests
	if count == 0 {
		return false, 0
	}
	for _, capture := range captures[startRequests:] {
		if capture.Status == http.StatusUnauthorized {
			return false, count
		}
	}
	return observer.Snapshot().Calls-startCalls == uint64(count), count
}

func e07UnauthenticatedProbe(hub *e07Hub, observer *e07ProviderObserver) (int, bool, string) {
	before := observer.Snapshot().Calls
	request, err := http.NewRequest(http.MethodPost, hub.baseURL+"/v1/responses", bytes.NewReader([]byte(`{"input":"unauthenticated"}`)))
	if err != nil {
		return 0, false, ""
	}
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return 0, false, ""
	}
	requestID := response.Header.Get("X-Request-Id")
	_ = response.Body.Close()
	after := observer.Snapshot().Calls
	return response.StatusCode, before == after, requestID
}

// e07UpstreamErrorProbe observes the direct provider status and the Hub
// response separately. It never compares bodies from independent provider
// calls; the deterministic test proves exact byte forwarding with a fixed
// upstream response.
func e07UpstreamErrorProbe(hub *e07Hub, gatewayKey string) e07UpstreamProbeObservation {
	observation := e07UpstreamProbeObservation{}
	payload := []byte(`{"model":"e07-intentionally-unknown-model","input":"Return one short response."}`)

	upstreamContext, cancelUpstream := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelUpstream()
	upstream, err := hub.provider.OpenTargetResponses(upstreamContext, providers.ResponsesTarget{ProviderID: providers.OpenCodeGoID}, payload, "application/json", "")
	if err != nil {
		return observation
	}
	observation.DirectStatus = upstream.StatusCode
	_, _ = e07ReadResponse(upstream)

	request, err := http.NewRequest(http.MethodPost, hub.baseURL+"/v1/responses", bytes.NewReader(payload))
	if err != nil {
		return observation
	}
	request.Header.Set("Authorization", "Bearer "+gatewayKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 2 * time.Minute}).Do(request)
	if err != nil {
		return observation
	}
	observation.HubStatus = response.StatusCode
	body, err := e07ReadResponse(response)
	if err != nil {
		return observation
	}
	observation.HubBodyRead = true
	observation.EnvelopePresent = e07ErrorEnvelopePresent(body)

	return observation
}
