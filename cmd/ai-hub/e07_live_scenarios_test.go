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
	setup := e07PrepareLiveSetup(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providerTransport, err := newE07ProviderTransport()
	if err != nil {
		t.Fatal("non-retrying provider transport could not be prepared")
	}
	defer providerTransport.CloseIdleConnections()
	observer := newE07ProviderObserver(providerTransport, e07ReplayMarker)
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
	if err := e07WriteCodexConfig(setup.CodexHome, hub.baseURL); err != nil {
		t.Fatal("isolated Codex configuration could not be written")
	}
	results := make(map[string]string)
	providerEnvironment := os.Environ()
	codexEnvironment := e07CodexEnvironment(os.Environ(), setup.CodexHome, setup.GatewayKey)
	if !e07EnvironmentHasKey(os.Environ(), config.EnvGatewayAPIKey) ||
		!e07EnvironmentHasKey(os.Environ(), config.EnvDataDir) ||
		!e07EnvironmentHasKey(providerEnvironment, config.EnvProviderOpenCodeGoAPIKey) ||
		!e07EnvironmentHasKey(codexEnvironment, e07ClientKeyEnv) ||
		e07EnvironmentHasKey(codexEnvironment, config.EnvProviderOpenCodeGoAPIKey) ||
		e07EnvironmentHasKey(codexEnvironment, config.EnvDataDir) ||
		e07EnvironmentHasKey(codexEnvironment, "OPENCODE_API_KEY") ||
		e07EnvironmentHasKey(codexEnvironment, config.EnvGatewayAPIKey) {
		t.Error("credential presence or process isolation check failed")
		results["credential_isolation"] = "FAIL"
	} else {
		results["credential_isolation"] = "PASS"
	}
	configBytes, err := os.ReadFile(setup.CodexHome + "/config.toml")
	if err != nil || bytes.Contains(configBytes, []byte(setup.GatewayKey)) || bytes.Contains(configBytes, []byte(setup.ProviderKey)) {
		t.Error("Codex configuration contains an unexpected credential")
		results["credential_isolation"] = "FAIL"
	}
	if _, err := os.Stat(setup.CodexHome); err != nil {
		t.Fatal("isolated CODEX_HOME was not created before Codex launch")
	}
	t.Logf("E07 metadata revision=%s go=%s codex=%s os=%s model=%s", setup.Revision, setup.GoVersion, setup.CodexVersion, setup.OSClass, e07ModelID)

	codexRoutingPassed := true
	codexResponseCount := 0
	beforeCalls := observer.Snapshot().Calls
	beforeRequests := len(hub.requestIDs.snapshot())
	basic := e07RunCodex(setup, setup.BasicDir, "Reply with exactly E07_BASIC_OK and no other text.", "exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
	results["basic_generation"] = e07CodexResult(basic, "E07_BASIC_OK")
	results["codex_progressive_rendering"] = e07ProgressiveResult(basic)
	passed, count := e07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
	codexRoutingPassed = codexRoutingPassed && passed
	codexResponseCount += count

	beforeCalls = observer.Snapshot().Calls
	beforeRequests = len(hub.requestIDs.snapshot())
	replayFirst := e07RunCodex(setup, setup.ReplayDir,
		"Remember the synthetic marker "+e07ReplayMarker+". Confirm only that you will remember it.",
		"exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
	var replaySecond e07CodexRun
	observer.inspectReplay.Store(true)
	if replayFirst.Ran && replayFirst.ExitCode == 0 && replayFirst.ThreadID != "" {
		replaySecond = e07RunCodex(setup, setup.ReplayDir,
			"Reply with exactly the synthetic marker from the previous turn and no other text.",
			"exec", "resume", "--json", replayFirst.ThreadID, "-")
	}
	observer.inspectReplay.Store(false)
	passed, count = e07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
	codexRoutingPassed = codexRoutingPassed && passed
	codexResponseCount += count
	replay := e07EvaluateReplay(replayFirst, replaySecond, count, e07ReplayExpectedRequests,
		observer.Snapshot().Replay, e07ReplayMarker)
	results["multi_turn_replay"] = replay.Result()
	t.Logf("%s", replay.Summary())

	beforeCalls = observer.Snapshot().Calls
	beforeRequests = len(hub.requestIDs.snapshot())
	tool := e07RunCodex(setup, setup.ToolDir,
		"Use your shell tool only in the current workspace. Write the exact marker "+e07ToolMarker+
			" to result.txt, read the file back, and reply with only its contents.",
		"exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
	toolResult, toolErr := os.ReadFile(setup.ToolDir + "/result.txt")
	if toolErr == nil && bytes.Equal(bytes.TrimSpace(toolResult), []byte(e07ToolMarker)) && strings.Contains(tool.Message, e07ToolMarker) {
		results["tool_round_trip"] = "PASS"
	} else if tool.EnvironmentLimit != "" {
		results["tool_round_trip"] = "NOT VERIFIED"
	} else {
		results["tool_round_trip"] = "FAIL"
	}
	passed, count = e07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
	codexRoutingPassed = codexRoutingPassed && passed
	codexResponseCount += count
	if codexRoutingPassed {
		results["codex_gateway_routing"] = "PASS"
	} else {
		results["codex_gateway_routing"] = "FAIL"
	}

	unauthStatus, providerUnchanged, unauthRequestID := e07UnauthenticatedProbe(hub, observer)
	if unauthStatus == http.StatusUnauthorized && providerUnchanged {
		results["gateway_authentication"] = "PASS"
	} else {
		results["gateway_authentication"] = "FAIL"
	}
	upstreamProbe := e07UpstreamErrorProbe(hub, setup.GatewayKey, observer)
	if upstreamProbe.PassthroughObserved() {
		results["upstream_error_passthrough"] = "PASS"
	} else {
		results["upstream_error_passthrough"] = "FAIL"
	}
	t.Logf("%s", upstreamProbe.Summary())
	stream, streamErr := e07MeasureSSE(hub.baseURL+"/v1/responses", setup.GatewayKey,
		[]byte(`{"model":"`+e07ModelID+`","input":"Stream a short numbered response progressively.","stream":true}`))
	if streamErr == nil && stream.Progressive {
		results["gateway_hop_streaming"] = "PASS"
	} else {
		results["gateway_hop_streaming"] = "FAIL"
	}

	// Every expected record comes from an E05 terminal observation keyed to a
	// captured request. List, correlation, outcome, schema, and by-ID checks
	// are independent; the post-restart check derives a fresh context.
	captures := hub.requestIDs.snapshot()
	expectations := e07BuildExpectedRecords(captures, hub.terminals.Snapshot(), []string{unauthRequestID})
	reader := &e07HTTPRecordReader{baseURL: hub.baseURL, gatewayKey: setup.GatewayKey}
	preStopContext, cancelPreStop := context.WithTimeout(context.Background(), 30*time.Second)
	preStop := e07AssertRecordsEventually(preStopContext, reader, expectations, 30*time.Second)
	cancelPreStop()
	results["execution_record_capture_query"] = e07ScenarioOutcome(preStop.Pass())
	t.Logf("%s", preStop.Summary("pre_stop"))

	stopErr := hub.stop()
	hubStopped = true
	if stopErr != nil {
		t.Error("AI Hub graceful shutdown failed")
		results["shutdown_restart_drain"] = "FAIL"
		t.Logf("E07 records phase=post_restart stop=FAIL")
	} else {
		restarted, err := startE07Hub(setup.Config, observer, logger)
		if err != nil {
			results["shutdown_restart_drain"] = "FAIL"
			t.Logf("E07 records phase=post_restart restart=FAIL")
		} else {
			restartedStopped := false
			t.Cleanup(func() {
				if !restartedStopped {
					if err := restarted.stop(); err != nil {
						t.Error("restarted AI Hub cleanup did not stop gracefully")
					}
				}
			})
			postReader := &e07HTTPRecordReader{baseURL: restarted.baseURL, gatewayKey: setup.GatewayKey}
			post := e07AssertRecordsAfterRestart(preStopContext, postReader, expectations, 30*time.Second)
			results["shutdown_restart_drain"] = e07ScenarioOutcome(post.Pass())
			t.Logf("%s", post.Summary("post_restart"))
			restartStopErr := restarted.stop()
			restartedStopped = true
			if restartStopErr != nil {
				t.Error("restarted AI Hub graceful shutdown failed")
				results["shutdown_restart_drain"] = "FAIL"
			}
		}
	}
	for name, result := range results {
		t.Logf("E07 scenario %s=%s", name, result)
		if e07ScenarioResultMustFail(name, result) {
			t.Errorf("E07 scenario failed: %s", name)
		}
	}
	t.Logf("E07 aggregate records_expected=%d records_matched=%d provider_calls=%d unauth_status=%d upstream_error_calls=%d stream_events=%d stream_first=%s stream_last=%s stream_complete=%s",
		preStop.Expected, preStop.Matched, observer.Snapshot().Calls, unauthStatus, upstreamProbe.ProviderCalls,
		stream.DataEvents, stream.FirstDataAt, stream.LastDataAt, stream.CompleteAt)
	t.Logf("E07 aggregate codex_responses=%d", codexResponseCount)
}
