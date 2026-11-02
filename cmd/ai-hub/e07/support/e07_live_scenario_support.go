package support

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	E07ToolMarker             = "E07-TOOL-MARKER-9A31"
	E07ReplayExpectedRequests = 2
)

// e07UpstreamProbeObservation keeps same-response boundary facts only.
type E07UpstreamProbeObservation struct {
	StatusPreserved      bool
	ContentTypePreserved bool
	BodyDigestMatches    bool
	BodyComplete         bool
	ClientErrorStatus    bool
	EnvelopePresent      bool
	ProviderCalls        uint64
}

// PassthroughObserved requires complete same-response observations at both
// sides of the Hub and a generic upstream error envelope.
func (observation E07UpstreamProbeObservation) PassthroughObserved() bool {
	return observation.ProviderCalls == 1 && observation.StatusPreserved && observation.ContentTypePreserved &&
		observation.BodyDigestMatches && observation.BodyComplete && observation.ClientErrorStatus &&
		observation.EnvelopePresent
}

func (observation E07UpstreamProbeObservation) Summary() string {
	return "E07 upstream_error status_preserved=" + strconv.FormatBool(observation.StatusPreserved) +
		" content_type_preserved=" + strconv.FormatBool(observation.ContentTypePreserved) +
		" body_digest_match=" + strconv.FormatBool(observation.BodyDigestMatches) +
		" body_complete=" + strconv.FormatBool(observation.BodyComplete) +
		" envelope=" + strconv.FormatBool(observation.EnvelopePresent) +
		" provider_calls=" + strconv.FormatUint(observation.ProviderCalls, 10) +
		" passthrough=" + strconv.FormatBool(observation.PassthroughObserved())
}

// e07ReplayDiagnostics separates every independent replay condition so a single
// aggregate FAIL can never hide which predicate failed.
type E07ReplayDiagnostics struct {
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

func E07EvaluateReplay(first, second E07CodexRun, requests, expectedRequests int, observation E07ReplayObservation, marker string) E07ReplayDiagnostics {
	diagnostics := E07ReplayDiagnostics{
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

func (diagnostics E07ReplayDiagnostics) Satisfied() bool {
	return diagnostics.FirstTurnOK && diagnostics.SecondTurnOK && diagnostics.RequestCountOK &&
		diagnostics.MarkerRecallOK && diagnostics.PriorContextOK && diagnostics.PreviousResponseIDAbsent
}

func (diagnostics E07ReplayDiagnostics) Result() string {
	if diagnostics.Satisfied() {
		return "PASS"
	}
	if diagnostics.EnvironmentLimited {
		return "NOT VERIFIED"
	}

	return "FAIL"
}

// Summary emits structural booleans and aggregate counts only. It never
// contains prompts, responses, identifiers, or payload bytes.
func (diagnostics E07ReplayDiagnostics) Summary() string {
	return "E07 replay first_turn=" + E07IntegrityResult(diagnostics.FirstTurnOK) +
		" second_turn=" + E07IntegrityResult(diagnostics.SecondTurnOK) +
		" requests=" + strconv.Itoa(diagnostics.Requests) +
		" expected_requests=" + strconv.Itoa(diagnostics.ExpectedRequests) +
		" request_count=" + E07IntegrityResult(diagnostics.RequestCountOK) +
		" marker_recall=" + E07IntegrityResult(diagnostics.MarkerRecallOK) +
		" prior_context=" + E07IntegrityResult(diagnostics.PriorContextOK) +
		" previous_response_id_absent=" + E07IntegrityResult(diagnostics.PreviousResponseIDAbsent) +
		" environment_limited=" + strconv.FormatBool(diagnostics.EnvironmentLimited)
}

func E07RunCodex(setup *E07LiveSetup, workspace, prompt string, args ...string) E07CodexRun {
	result, err := RunE07Codex(setup.CodexPath, setup.CodexHome, setup.GatewayKey, workspace, prompt, args...)
	if err != nil && result.EnvironmentLimit == "" {
		result.ExitCode = 1
	}
	return result
}

func E07CodexResult(result E07CodexRun, expected string) string {
	if result.Ran && result.ExitCode == 0 && strings.Contains(result.Message, expected) {
		return "PASS"
	}
	return "FAIL"
}

func E07ProgressiveResult(result E07CodexRun) string {
	if result.ProgressiveSeen {
		return "PASS"
	}
	return "NOT VERIFIED"
}

func E07ScenarioOutcome(passed bool) string {
	if passed {
		return "PASS"
	}

	return "FAIL"
}

func E07ScenarioResultMustFail(name, result string) bool {
	if result == "PASS" {
		return false
	}
	return name != "codex_progressive_rendering" || result != "NOT VERIFIED"
}

func E07CodexRequestsReachHub(startRequests int, startCalls uint64, hub *E07Hub, observer *E07ProviderObserver) (bool, int) {
	captures := hub.RequestIDs.Snapshot()
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

func E07UnauthenticatedProbe(hub *E07Hub, observer *E07ProviderObserver) (int, bool, string) {
	before := observer.Snapshot().Calls
	request, err := http.NewRequest(http.MethodPost, hub.BaseURL+"/v1/responses", bytes.NewReader([]byte(`{"input":"unauthenticated"}`)))
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

// e07UpstreamErrorProbe observes one response at the provider and Hub client
// boundaries. It retains body digests in memory only until the comparison.
func E07UpstreamErrorProbe(hub *E07Hub, gatewayKey string, observer *E07ProviderObserver) (probe E07UpstreamProbeObservation) {
	callsBefore := observer.Snapshot().Calls
	defer func() { probe.ProviderCalls = observer.Snapshot().Calls - callsBefore }()
	payload := []byte(`{"model":"e07-intentionally-unknown-model","input":"Return one short response."}`)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, hub.BaseURL+"/v1/responses", bytes.NewReader(payload))
	if err != nil {
		return probe
	}
	request.Header.Set("Authorization", "Bearer "+gatewayKey)
	request.Header.Set("Content-Type", "application/json")
	observer.ArmResponseFidelityObservation()
	httpClient := &http.Client{
		Timeout:       2 * time.Minute,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := httpClient.Do(request)
	if err != nil {
		_ = observer.TakeResponseFidelityObservation()
		return probe
	}
	body, err := E07ReadResponse(response)
	client := E07ClientResponseObservation(response, body, err)
	upstream := observer.TakeResponseFidelityObservation()
	probe.StatusPreserved = upstream.StatusCode == client.StatusCode
	probe.ContentTypePreserved = upstream.ContentType == client.ContentType
	probe.BodyComplete = upstream.BodyComplete && client.BodyComplete
	probe.BodyDigestMatches = probe.BodyComplete && upstream.BodyDigest == client.BodyDigest
	probe.ClientErrorStatus = client.StatusCode >= http.StatusBadRequest && client.StatusCode < http.StatusInternalServerError
	if err == nil {
		probe.EnvelopePresent = E07ErrorEnvelopePresent(body)
	}
	clear(body)
	return probe
}
