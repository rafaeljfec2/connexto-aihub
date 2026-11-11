package support

import (
	"os"
	"testing"
	"time"
)

const (
	E07LiveRequestTimeout        = 2 * time.Minute
	E07ExpectedLiveProviderCalls = 8
)

type E07LiveScenarioEvidence struct {
	StartedAt   time.Time
	CallsBefore uint64
}

func E07StartLiveScenario(observer *E07ProviderObserver) E07LiveScenarioEvidence {
	return E07LiveScenarioEvidence{StartedAt: time.Now().UTC(), CallsBefore: E07ProviderCalls(observer)}
}

func E07FinishLiveScenario(t *testing.T, observer *E07ProviderObserver, evidence E07LiveScenarioEvidence, name, result string) uint64 {
	t.Helper()
	callsAfter := E07ProviderCalls(observer)
	endedAt := time.Now().UTC()
	t.Logf("E07 scenario id=%s started_at=%s ended_at=%s provider_calls_before=%d provider_calls_after=%d result=%s",
		name, evidence.StartedAt.Format(time.RFC3339Nano), endedAt.Format(time.RFC3339Nano), evidence.CallsBefore, callsAfter, result)
	if observer != nil && observer.BudgetSnapshot().Rejected > 0 {
		t.Fatalf("E07 provider-call budget rejected a request before transport: %s", name)
	}
	if E07ScenarioResultMustFail(name, result) {
		t.Errorf("E07 scenario failed: %s", name)
	}
	return callsAfter
}

func E07RunLiveScenario(t *testing.T, observer *E07ProviderObserver, name string, run func() string) string {
	t.Helper()
	evidence := E07StartLiveScenario(observer)
	t.Logf("E07 scenario id=%s started_at=%s provider_calls_before=%d", name,
		evidence.StartedAt.Format(time.RFC3339Nano), evidence.CallsBefore)
	result := run()
	integrityPass := observer == nil || observer.Integrity == nil || observer.Integrity.Check(t)
	if !integrityPass {
		result = "FAIL"
	}
	E07FinishLiveScenario(t, observer, evidence, name, result)
	if !integrityPass {
		t.Fatalf("E07 integrity checkpoint failed; remaining provider traffic stopped: %s", name)
	}
	return result
}

func E07RunLiveCleanupScenario(t *testing.T, observer *E07ProviderObserver, name string, run func() string) {
	t.Helper()
	evidence := E07StartLiveScenario(observer)
	t.Logf("E07 scenario id=%s started_at=%s provider_calls_before=%d", name,
		evidence.StartedAt.Format(time.RFC3339Nano), evidence.CallsBefore)
	result := run()
	callsAfter := E07ProviderCalls(observer)
	endedAt := time.Now().UTC()
	t.Logf("E07 scenario id=%s started_at=%s ended_at=%s provider_calls_before=%d provider_calls_after=%d result=%s",
		name, evidence.StartedAt.Format(time.RFC3339Nano), endedAt.Format(time.RFC3339Nano), evidence.CallsBefore, callsAfter, result)
	if result != "PASS" || (observer != nil && observer.BudgetSnapshot().Rejected > 0) {
		t.Error("E07 cleanup integrity scenario failed")
	}
}

func E07ProviderCalls(observer *E07ProviderObserver) uint64 {
	if observer == nil {
		return 0
	}
	return observer.Snapshot().Calls
}

type E07LiveIntegrityState struct {
	Root                  string
	TempRoot              string
	ManifestDirectory     string
	BeforeEnvironment     map[string]string
	BeforeCodexManifest   string
	BeforePrivateManifest string
	PrivateCodexHome      string
	ManifestPath          string
}

type E07LiveIntegrityGuard struct {
	Root               string
	PrivateCodexHome   string
	BaselineCodex      []E07CodexTreeEntry
	BaselineRepository E07IntegritySnapshot
}

func (guard *E07LiveIntegrityGuard) Check(t *testing.T) bool {
	t.Helper()
	boundaryErr := E07RequirePrivateCodexMount(os.Getenv(E07PrivateRootEnv))
	currentCodex, codexErr := E07SnapshotCodexTree()
	codexPass := boundaryErr == nil && codexErr == nil && E07CodexTreeUnchanged(guard.BaselineCodex, currentCodex)
	privateState, privateErr := E07SnapshotPrivateCodexState(guard.PrivateCodexHome)
	privatePass := privateErr == nil && E07PrivateCodexStateSnapshotValid(
		privateState, len(E07CodexRuntimeDirectories(guard.PrivateCodexHome)))
	currentRepository, repositoryErr := E07SnapshotRepository(guard.Root)
	repositoryPass := repositoryErr == nil
	if repositoryPass {
		comparison := E07CompareIntegrity(guard.BaselineRepository, currentRepository)
		repositoryPass = comparison.Tree && comparison.Env && comparison.Tracked && comparison.GitState
	}
	t.Logf("E07 integrity checkpoint isolation=%s private_state=%s repository=%s",
		E07IntegrityResult(codexPass), E07IntegrityResult(privatePass), E07IntegrityResult(repositoryPass))
	return codexPass && privatePass && repositoryPass
}

func E07FinalizeLiveIntegrity(t *testing.T, state E07LiveIntegrityState) string {
	t.Helper()
	E07RestoreEnvironment(state.BeforeEnvironment)
	afterCodex, codexErr := E07SnapshotCodexTree()
	afterCodexManifest, saveErr := E07SaveCodexTreeManifest(state.ManifestDirectory, "after", afterCodex)
	beforeCodex, beforeErr := E07LoadCodexTreeManifest(state.BeforeCodexManifest)
	afterCodexSaved, afterErr := E07LoadCodexTreeManifest(afterCodexManifest)
	codexPass := codexErr == nil && saveErr == nil && beforeErr == nil && afterErr == nil &&
		E07CodexTreeUnchanged(beforeCodex, afterCodexSaved)
	t.Logf("E07 codex_tree=%s links=%d", E07IntegrityResult(codexPass), E07CodexTreeLinkCount(afterCodexSaved))
	afterPrivate, privateErr := E07SnapshotPrivateCodexState(state.PrivateCodexHome)
	afterPrivateManifest, privateSaveErr := E07SaveCodexTreeManifest(state.ManifestDirectory, "private-after", afterPrivate)
	beforePrivate, privateBeforeErr := E07LoadCodexTreeManifest(state.BeforePrivateManifest)
	afterPrivateSaved, privateAfterErr := E07LoadCodexTreeManifest(afterPrivateManifest)
	privatePass := privateErr == nil && privateSaveErr == nil && privateBeforeErr == nil && privateAfterErr == nil &&
		E07PrivateCodexStateSnapshotValid(beforePrivate, len(E07CodexRuntimeDirectories(state.PrivateCodexHome))) &&
		E07PrivateCodexStateSnapshotValid(afterPrivateSaved, len(E07CodexRuntimeDirectories(state.PrivateCodexHome)))
	t.Logf("E07 private_codex_state=%s roots_before=%d roots_after=%d", E07IntegrityResult(privatePass),
		E07CodexTreeRootCount(beforePrivate), E07CodexTreeRootCount(afterPrivateSaved))

	beforeRepository, manifestErr := E07LoadIntegrityManifest(state.ManifestPath)
	afterRepository, repositoryErr := E07SnapshotRepository(state.Root)
	repositoryPass := false
	if manifestErr != nil || repositoryErr != nil {
		t.Log("E07 environment_integrity repository_snapshot=UNVERIFIED")
	} else {
		comparison := E07CompareIntegrity(beforeRepository, afterRepository)
		repositoryPass = comparison.Tree && comparison.Env && comparison.Tracked && comparison.GitState
		t.Logf("E07 environment_integrity %s", comparison.Summary())
	}
	if err := os.RemoveAll(state.TempRoot); err != nil {
		t.Log("E07 isolated validation cleanup=FAIL")
		return "FAIL"
	}
	if codexPass && privatePass && repositoryPass {
		return "PASS"
	}
	return "FAIL"
}
