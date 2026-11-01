package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
)

func TestE07LiveCodexOpenCodeGo(t *testing.T) {
	if os.Getenv("AIHUB_E07_LIVE") != "1" {
		t.Skip("set AIHUB_E07_LIVE=1 to run the bounded real-provider scenario")
	}
	passStarted := time.Now()
	var observer *e07ProviderObserver
	t.Cleanup(func() {
		ended := time.Now()
		budget := e07ProviderBudgetSnapshot{}
		if observer != nil {
			budget = observer.BudgetSnapshot()
		}
		t.Logf("E07 live_pass started_at=%s ended_at=%s elapsed=%s provider_calls=%d attempted=%d rejected=%d forwarded=%d",
			passStarted.UTC().Format(time.RFC3339Nano), ended.UTC().Format(time.RFC3339Nano), ended.Sub(passStarted),
			e07ProviderCalls(observer), budget.Attempted, budget.Rejected, budget.Forwarded)
	})
	setup := e07PrepareLiveSetup(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providerTransport, err := newE07ProviderTransport()
	if err != nil {
		t.Fatal("non-retrying provider transport could not be prepared")
	}
	defer providerTransport.CloseIdleConnections()
	observer = newE07ProviderObserver(providerTransport, e07ReplayMarker)
	observer.integrity = setup.integrity
	setup.observer = observer
	setup.Config.Catalog.DataDir = setup.DataDir
	if err := setup.Config.Validate(); err != nil {
		t.Fatal("isolated AI Hub configuration is invalid")
	}
	hub, err := startE07Hub(setup.Config, observer, logger)
	if err != nil {
		t.Fatal("AI Hub could not start with the configured provider")
	}
	hubStopped := false
	t.Cleanup(func() {
		if !hubStopped {
			if err := hub.stop(); err != nil {
				t.Error("AI Hub cleanup did not stop gracefully")
			}
		}
	})
	codexEnvironment := e07CodexEnvironment(os.Environ(), setup.CodexHome, setup.GatewayKey)
	if err := e07WriteCodexConfig(setup.CodexHome, hub.baseURL, codexEnvironment); err != nil {
		t.Fatal("isolated Codex configuration could not be written")
	}
	providerEnvironment := os.Environ()
	e07RunLiveScenario(t, observer, "credential_isolation_preflight", func() string {
		if !e07EnvironmentHasKey(os.Environ(), config.EnvGatewayAPIKey) ||
			!e07EnvironmentHasKey(os.Environ(), config.EnvDataDir) ||
			!e07EnvironmentHasKey(providerEnvironment, config.EnvProviderOpenCodeGoAPIKey) ||
			!e07EnvironmentHasKey(codexEnvironment, e07ClientKeyEnv) ||
			e07EnvironmentHasKey(codexEnvironment, config.EnvProviderOpenCodeGoAPIKey) ||
			e07EnvironmentHasKey(codexEnvironment, config.EnvDataDir) ||
			e07EnvironmentHasKey(codexEnvironment, "OPENCODE_API_KEY") ||
			e07EnvironmentHasKey(codexEnvironment, config.EnvGatewayAPIKey) {
			return "FAIL"
		}
		configBytes, err := os.ReadFile(setup.CodexHome + "/config.toml")
		if err != nil || bytes.Contains(configBytes, []byte(setup.GatewayKey)) || bytes.Contains(configBytes, []byte(setup.ProviderKey)) {
			return "FAIL"
		}
		return "PASS"
	})
	if _, err := os.Stat(setup.CodexHome); err != nil {
		t.Fatal("isolated CODEX_HOME was not created before Codex launch")
	}
	t.Logf("E07 metadata revision=%s go=%s codex=%s os=%s model=%s", setup.Revision, setup.GoVersion, setup.CodexVersion, setup.OSClass, e07ModelID)

	codexResponseCount := 0
	var unauthRequestID string
	var unauthStatus int
	var upstreamProbe e07UpstreamProbeObservation
	var stream e07SSEObservation
	e07RunLiveScenario(t, observer, "basic_generation", func() string {
		beforeCalls := observer.Snapshot().Calls
		beforeRequests := len(hub.requestIDs.snapshot())
		basic := e07RunCodex(setup, setup.BasicDir, "Reply with exactly E07_BASIC_OK and no other text.", "exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
		passed, count := e07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
		codexResponseCount += count
		if !passed {
			return "FAIL"
		}
		progressive := e07ProgressiveResult(basic)
		t.Logf("E07 scenario id=codex_progressive_rendering result=%s", progressive)
		return e07CodexResult(basic, "E07_BASIC_OK")
	})
	e07RunLiveScenario(t, observer, "gateway_authentication", func() string {
		var providerUnchanged bool
		unauthStatus, providerUnchanged, unauthRequestID = e07UnauthenticatedProbe(hub, observer)
		if unauthStatus == http.StatusUnauthorized && providerUnchanged {
			return "PASS"
		}
		return "FAIL"
	})
	e07RunLiveScenario(t, observer, "multi_turn_replay", func() string {
		beforeCalls := observer.Snapshot().Calls
		beforeRequests := len(hub.requestIDs.snapshot())
		replayFirst := e07RunCodex(setup, setup.ReplayDir,
			"Remember the synthetic marker "+e07ReplayMarker+". Confirm only that you will remember it.",
			"exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
		var replaySecond e07CodexRun
		observer.inspectReplay.Store(true)
		if replayFirst.Ran && replayFirst.ExitCode == 0 && replayFirst.ThreadID != "" {
			replaySecond = e07RunCodex(setup, setup.ReplayDir,
				"Reply with exactly the synthetic marker from the previous turn and no other text.",
				"exec", "resume", "--json", "--skip-git-repo-check", replayFirst.ThreadID, "-")
		}
		observer.inspectReplay.Store(false)
		passed, count := e07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
		codexResponseCount += count
		replay := e07EvaluateReplay(replayFirst, replaySecond, count, e07ReplayExpectedRequests,
			observer.Snapshot().Replay, e07ReplayMarker)
		t.Logf("%s", replay.Summary())
		if !passed {
			return "FAIL"
		}
		return replay.Result()
	})
	e07RunLiveScenario(t, observer, "sandboxed_tool_calling", func() string {
		beforeCalls := observer.Snapshot().Calls
		beforeRequests := len(hub.requestIDs.snapshot())
		tool := e07RunCodex(setup, setup.ToolDir,
			"Use your shell tool only in the current workspace. Write the exact marker "+e07ToolMarker+
				" to result.txt, read the file back, and reply with only its contents.",
			"exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
		toolResult, toolErr := os.ReadFile(setup.ToolDir + "/result.txt")
		passed, count := e07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
		codexResponseCount += count
		if !passed {
			return "FAIL"
		}
		if toolErr == nil && bytes.Equal(bytes.TrimSpace(toolResult), []byte(e07ToolMarker)) && strings.Contains(tool.Message, e07ToolMarker) {
			return "PASS"
		}
		return "FAIL"
	})
	e07RunLiveScenario(t, observer, "upstream_error_passthrough", func() string {
		upstreamProbe = e07UpstreamErrorProbe(hub, setup.GatewayKey, observer)
		t.Logf("%s", upstreamProbe.Summary())
		if upstreamProbe.PassthroughObserved() {
			return "PASS"
		}
		return "FAIL"
	})
	e07RunLiveScenario(t, observer, "gateway_streaming", func() string {
		var streamErr error
		stream, streamErr = e07MeasureSSE(hub.baseURL+"/v1/responses", setup.GatewayKey,
			[]byte(`{"model":"`+e07ModelID+`","input":"Stream a short numbered response progressively.","stream":true}`))
		if streamErr == nil && stream.Progressive {
			return "PASS"
		}
		return "FAIL"
	})

	// Every expected record comes from an E05 terminal observation keyed to a
	// captured request. List, correlation, outcome, schema, and by-ID checks
	// are independent; the post-restart check derives a fresh context.
	captures := hub.requestIDs.snapshot()
	expectations := e07BuildExpectedRecords(captures, hub.terminals.Snapshot(), []string{unauthRequestID})
	reader := &e07HTTPRecordReader{baseURL: hub.baseURL, gatewayKey: setup.GatewayKey}
	var preStop e07RecordAssertions
	e07RunLiveScenario(t, observer, "execution_record_listing_and_by_id", func() string {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		preStop = e07AssertRecordsEventually(ctx, reader, expectations, 30*time.Second)
		t.Logf("%s", preStop.Summary("pre_stop"))
		return e07ScenarioOutcome(preStop.Pass())
	})
	e07RunLiveScenario(t, observer, "shutdown_restart_recovery", func() string {
		stopErr := hub.stop()
		hubStopped = true
		if stopErr != nil {
			return "FAIL"
		}
		restarted, err := startE07Hub(setup.Config, observer, logger)
		if err != nil {
			return "FAIL"
		}
		restartedStopped := false
		t.Cleanup(func() {
			if !restartedStopped {
				if err := restarted.stop(); err != nil {
					t.Error("restarted AI Hub cleanup did not stop gracefully")
				}
			}
		})
		postReader := &e07HTTPRecordReader{baseURL: restarted.baseURL, gatewayKey: setup.GatewayKey}
		post := e07AssertRecordsAfterRestart(context.Background(), postReader, expectations, 30*time.Second)
		t.Logf("%s", post.Summary("post_restart"))
		restartStopErr := restarted.stop()
		restartedStopped = true
		if restartStopErr != nil {
			return "FAIL"
		}
		return e07ScenarioOutcome(post.Pass())
	})
	if got := observer.Snapshot().Calls; got < e07ExpectedLiveProviderCalls || got > e07ProviderCallLimit {
		t.Fatalf("E07 live provider calls=%d, expected at least %d and at most %d",
			got, e07ExpectedLiveProviderCalls, e07ProviderCallLimit)
	}
	t.Logf("E07 aggregate records_expected=%d records_matched=%d provider_calls=%d expected_provider_calls=%d maximum_provider_calls=%d unauth_status=%d upstream_error_calls=%d stream_events=%d stream_first=%s stream_last=%s stream_complete=%s",
		preStop.Expected, preStop.Matched, observer.Snapshot().Calls, e07ExpectedLiveProviderCalls, e07ProviderCallLimit, unauthStatus, upstreamProbe.ProviderCalls,
		stream.DataEvents, stream.FirstDataAt, stream.LastDataAt, stream.CompleteAt)
	t.Logf("E07 aggregate codex_responses=%d", codexResponseCount)
}
