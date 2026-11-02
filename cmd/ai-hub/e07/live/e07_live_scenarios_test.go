package live

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"aihub/internal/config"
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE07LiveCodexOpenCodeGo(t *testing.T) {
	if os.Getenv("AIHUB_E07_LIVE") != "1" {
		t.Skip("set AIHUB_E07_LIVE=1 to run the bounded real-provider scenario")
	}
	passStarted := time.Now()
	var observer *e07support.E07ProviderObserver
	t.Cleanup(func() {
		ended := time.Now()
		budget := e07support.E07ProviderBudgetSnapshot{}
		if observer != nil {
			budget = observer.BudgetSnapshot()
		}
		t.Logf("E07 live_pass started_at=%s ended_at=%s elapsed=%s provider_calls=%d attempted=%d rejected=%d forwarded=%d",
			passStarted.UTC().Format(time.RFC3339Nano), ended.UTC().Format(time.RFC3339Nano), ended.Sub(passStarted),
			e07support.E07ProviderCalls(observer), budget.Attempted, budget.Rejected, budget.Forwarded)
	})
	setup := e07support.E07PrepareLiveSetup(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providerTransport, err := e07support.NewE07ProviderTransport()
	if err != nil {
		t.Fatal("non-retrying provider transport could not be prepared")
	}
	defer providerTransport.CloseIdleConnections()
	observer = e07support.NewE07ProviderObserver(providerTransport, e07support.E07ReplayMarker)
	observer.Integrity = setup.Integrity
	setup.Observer = observer
	setup.Config.Catalog.DataDir = setup.DataDir
	if err := setup.Config.Validate(); err != nil {
		t.Fatal("isolated AI Hub configuration is invalid")
	}
	hub, err := e07support.StartE07Hub(setup.Config, observer, logger)
	if err != nil {
		t.Fatal("AI Hub could not start with the configured provider")
	}
	hubStopped := false
	t.Cleanup(func() {
		if !hubStopped {
			if err := hub.Stop(); err != nil {
				t.Error("AI Hub cleanup did not stop gracefully")
			}
		}
	})
	codexEnvironment := e07support.E07CodexEnvironment(os.Environ(), setup.CodexHome, setup.GatewayKey)
	if err := e07support.E07WriteCodexConfig(setup.CodexHome, hub.BaseURL, codexEnvironment); err != nil {
		t.Fatal("isolated Codex configuration could not be written")
	}
	providerEnvironment := os.Environ()
	e07support.E07RunLiveScenario(t, observer, "credential_isolation_preflight", func() string {
		if !e07support.E07EnvironmentHasKey(os.Environ(), config.EnvGatewayAPIKey) ||
			!e07support.E07EnvironmentHasKey(os.Environ(), config.EnvDataDir) ||
			!e07support.E07EnvironmentHasKey(providerEnvironment, config.EnvProviderOpenCodeGoAPIKey) ||
			!e07support.E07EnvironmentHasKey(codexEnvironment, e07support.E07ClientKeyEnv) ||
			e07support.E07EnvironmentHasKey(codexEnvironment, config.EnvProviderOpenCodeGoAPIKey) ||
			e07support.E07EnvironmentHasKey(codexEnvironment, config.EnvDataDir) ||
			e07support.E07EnvironmentHasKey(codexEnvironment, "OPENCODE_API_KEY") ||
			e07support.E07EnvironmentHasKey(codexEnvironment, config.EnvGatewayAPIKey) {
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
	t.Logf("E07 metadata revision=%s go=%s codex=%s os=%s model=%s", setup.Revision, setup.GoVersion, setup.CodexVersion, setup.OSClass, e07support.E07ModelID)

	codexResponseCount := 0
	var unauthRequestID string
	var unauthStatus int
	var upstreamProbe e07support.E07UpstreamProbeObservation
	var stream e07support.E07SSEObservation
	e07support.E07RunLiveScenario(t, observer, "basic_generation", func() string {
		beforeCalls := observer.Snapshot().Calls
		beforeRequests := len(hub.RequestIDs.Snapshot())
		basic := e07support.E07RunCodex(setup, setup.BasicDir, "Reply with exactly E07_BASIC_OK and no other text.", "exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
		passed, count := e07support.E07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
		codexResponseCount += count
		if !passed {
			return "FAIL"
		}
		progressive := e07support.E07ProgressiveResult(basic)
		t.Logf("E07 scenario id=codex_progressive_rendering result=%s", progressive)
		return e07support.E07CodexResult(basic, "E07_BASIC_OK")
	})
	e07support.E07RunLiveScenario(t, observer, "gateway_authentication", func() string {
		var providerUnchanged bool
		unauthStatus, providerUnchanged, unauthRequestID = e07support.E07UnauthenticatedProbe(hub, observer)
		if unauthStatus == http.StatusUnauthorized && providerUnchanged {
			return "PASS"
		}
		return "FAIL"
	})
	e07support.E07RunLiveScenario(t, observer, "multi_turn_replay", func() string {
		beforeCalls := observer.Snapshot().Calls
		beforeRequests := len(hub.RequestIDs.Snapshot())
		replayFirst := e07support.E07RunCodex(setup, setup.ReplayDir,
			"Remember the synthetic marker "+e07support.E07ReplayMarker+". Confirm only that you will remember it.",
			"exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
		var replaySecond e07support.E07CodexRun
		observer.InspectReplay.Store(true)
		if replayFirst.Ran && replayFirst.ExitCode == 0 && replayFirst.ThreadID != "" {
			replaySecond = e07support.E07RunCodex(setup, setup.ReplayDir,
				"Reply with exactly the synthetic marker from the previous turn and no other text.",
				"exec", "resume", "--json", "--skip-git-repo-check", replayFirst.ThreadID, "-")
		}
		observer.InspectReplay.Store(false)
		passed, count := e07support.E07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
		codexResponseCount += count
		replay := e07support.E07EvaluateReplay(replayFirst, replaySecond, count, e07support.E07ReplayExpectedRequests,
			observer.Snapshot().Replay, e07support.E07ReplayMarker)
		t.Logf("%s", replay.Summary())
		if !passed {
			return "FAIL"
		}
		return replay.Result()
	})
	e07support.E07RunLiveScenario(t, observer, "sandboxed_tool_calling", func() string {
		beforeCalls := observer.Snapshot().Calls
		beforeRequests := len(hub.RequestIDs.Snapshot())
		tool := e07support.E07RunCodex(setup, setup.ToolDir,
			"Use your shell tool only in the current workspace. Write the exact marker "+e07support.E07ToolMarker+
				" to result.txt, read the file back, and reply with only its contents.",
			"exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
		toolResult, toolErr := os.ReadFile(setup.ToolDir + "/result.txt")
		passed, count := e07support.E07CodexRequestsReachHub(beforeRequests, beforeCalls, hub, observer)
		codexResponseCount += count
		if !passed {
			return "FAIL"
		}
		if toolErr == nil && bytes.Equal(bytes.TrimSpace(toolResult), []byte(e07support.E07ToolMarker)) && strings.Contains(tool.Message, e07support.E07ToolMarker) {
			return "PASS"
		}
		return "FAIL"
	})
	e07support.E07RunLiveScenario(t, observer, "upstream_error_passthrough", func() string {
		upstreamProbe = e07support.E07UpstreamErrorProbe(hub, setup.GatewayKey, observer)
		t.Logf("%s", upstreamProbe.Summary())
		if upstreamProbe.PassthroughObserved() {
			return "PASS"
		}
		return "FAIL"
	})
	e07support.E07RunLiveScenario(t, observer, "gateway_streaming", func() string {
		var streamErr error
		stream, streamErr = e07support.E07MeasureSSE(hub.BaseURL+"/v1/responses", setup.GatewayKey,
			[]byte(`{"model":"`+e07support.E07ModelID+`","input":"Stream a short numbered response progressively.","stream":true}`))
		if streamErr == nil && stream.Progressive {
			return "PASS"
		}
		return "FAIL"
	})

	// Every expected record comes from an E05 terminal observation keyed to a
	// captured request. List, correlation, outcome, schema, and by-ID checks
	// are independent; the post-restart check derives a fresh context.
	captures := hub.RequestIDs.Snapshot()
	expectations := e07support.E07BuildExpectedRecords(captures, hub.Terminals.Snapshot(), []string{unauthRequestID})
	reader := &e07support.E07HTTPRecordReader{BaseURL: hub.BaseURL, GatewayKey: setup.GatewayKey}
	var preStop e07support.E07RecordAssertions
	e07support.E07RunLiveScenario(t, observer, "execution_record_listing_and_by_id", func() string {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		preStop = e07support.E07AssertRecordsEventually(ctx, reader, expectations, 30*time.Second)
		t.Logf("%s", preStop.Summary("pre_stop"))
		return e07support.E07ScenarioOutcome(preStop.Pass())
	})
	e07support.E07RunLiveScenario(t, observer, "shutdown_restart_recovery", func() string {
		stopErr := hub.Stop()
		hubStopped = true
		if stopErr != nil {
			return "FAIL"
		}
		restarted, err := e07support.StartE07Hub(setup.Config, observer, logger)
		if err != nil {
			return "FAIL"
		}
		restartedStopped := false
		t.Cleanup(func() {
			if !restartedStopped {
				if err := restarted.Stop(); err != nil {
					t.Error("restarted AI Hub cleanup did not stop gracefully")
				}
			}
		})
		postReader := &e07support.E07HTTPRecordReader{BaseURL: restarted.BaseURL, GatewayKey: setup.GatewayKey}
		post := e07support.E07AssertRecordsAfterRestart(context.Background(), postReader, expectations, 30*time.Second)
		t.Logf("%s", post.Summary("post_restart"))
		restartStopErr := restarted.Stop()
		restartedStopped = true
		if restartStopErr != nil {
			return "FAIL"
		}
		return e07support.E07ScenarioOutcome(post.Pass())
	})
	if got := observer.Snapshot().Calls; got < e07support.E07ExpectedLiveProviderCalls || got > e07support.E07ProviderCallLimit {
		t.Fatalf("E07 live provider calls=%d, expected at least %d and at most %d",
			got, e07support.E07ExpectedLiveProviderCalls, e07support.E07ProviderCallLimit)
	}
	t.Logf("E07 aggregate records_expected=%d records_matched=%d provider_calls=%d expected_provider_calls=%d maximum_provider_calls=%d unauth_status=%d upstream_error_calls=%d stream_events=%d stream_first=%s stream_last=%s stream_complete=%s",
		preStop.Expected, preStop.Matched, observer.Snapshot().Calls, e07support.E07ExpectedLiveProviderCalls, e07support.E07ProviderCallLimit, unauthStatus, upstreamProbe.ProviderCalls,
		stream.DataEvents, stream.FirstDataAt, stream.LastDataAt, stream.CompleteAt)
	t.Logf("E07 aggregate codex_responses=%d", codexResponseCount)
}
