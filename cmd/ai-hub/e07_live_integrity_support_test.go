package main

import (
	"os"
	"testing"
)

type e07LiveIntegrityState struct {
	root                  string
	tempRoot              string
	manifestDirectory     string
	beforeEnvironment     map[string]string
	beforeCodexManifest   string
	beforePrivateManifest string
	privateCodexHome      string
	manifestPath          string
}

type e07LiveIntegrityGuard struct {
	root               string
	privateCodexHome   string
	baselineCodex      []e07CodexTreeEntry
	baselineRepository e07IntegritySnapshot
}

func (guard *e07LiveIntegrityGuard) Check(t *testing.T) bool {
	t.Helper()
	boundaryErr := e07RequirePrivateCodexMount(os.Getenv(e07PrivateRootEnv))
	currentCodex, codexErr := e07SnapshotCodexTree()
	codexPass := boundaryErr == nil && codexErr == nil && e07CodexTreeUnchanged(guard.baselineCodex, currentCodex)
	privateState, privateErr := e07SnapshotPrivateCodexState(guard.privateCodexHome)
	privatePass := privateErr == nil && e07PrivateCodexStateSnapshotValid(
		privateState, len(e07CodexRuntimeDirectories(guard.privateCodexHome)))
	currentRepository, repositoryErr := e07SnapshotRepository(guard.root)
	repositoryPass := repositoryErr == nil
	if repositoryPass {
		comparison := e07CompareIntegrity(guard.baselineRepository, currentRepository)
		repositoryPass = comparison.Tree && comparison.Env && comparison.Tracked && comparison.GitState
	}
	t.Logf("E07 integrity checkpoint isolation=%s private_state=%s repository=%s",
		e07IntegrityResult(codexPass), e07IntegrityResult(privatePass), e07IntegrityResult(repositoryPass))
	return codexPass && privatePass && repositoryPass
}

func e07FinalizeLiveIntegrity(t *testing.T, state e07LiveIntegrityState) string {
	t.Helper()
	e07RestoreEnvironment(state.beforeEnvironment)
	afterCodex, codexErr := e07SnapshotCodexTree()
	afterCodexManifest, saveErr := e07SaveCodexTreeManifest(state.manifestDirectory, "after", afterCodex)
	beforeCodex, beforeErr := e07LoadCodexTreeManifest(state.beforeCodexManifest)
	afterCodexSaved, afterErr := e07LoadCodexTreeManifest(afterCodexManifest)
	codexPass := codexErr == nil && saveErr == nil && beforeErr == nil && afterErr == nil &&
		e07CodexTreeUnchanged(beforeCodex, afterCodexSaved)
	t.Logf("E07 codex_tree=%s links=%d", e07IntegrityResult(codexPass), e07CodexTreeLinkCount(afterCodexSaved))
	afterPrivate, privateErr := e07SnapshotPrivateCodexState(state.privateCodexHome)
	afterPrivateManifest, privateSaveErr := e07SaveCodexTreeManifest(state.manifestDirectory, "private-after", afterPrivate)
	beforePrivate, privateBeforeErr := e07LoadCodexTreeManifest(state.beforePrivateManifest)
	afterPrivateSaved, privateAfterErr := e07LoadCodexTreeManifest(afterPrivateManifest)
	privatePass := privateErr == nil && privateSaveErr == nil && privateBeforeErr == nil && privateAfterErr == nil &&
		e07PrivateCodexStateSnapshotValid(beforePrivate, len(e07CodexRuntimeDirectories(state.privateCodexHome))) &&
		e07PrivateCodexStateSnapshotValid(afterPrivateSaved, len(e07CodexRuntimeDirectories(state.privateCodexHome)))
	t.Logf("E07 private_codex_state=%s roots_before=%d roots_after=%d", e07IntegrityResult(privatePass),
		e07CodexTreeRootCount(beforePrivate), e07CodexTreeRootCount(afterPrivateSaved))

	beforeRepository, manifestErr := e07LoadIntegrityManifest(state.manifestPath)
	afterRepository, repositoryErr := e07SnapshotRepository(state.root)
	repositoryPass := false
	if manifestErr != nil || repositoryErr != nil {
		t.Log("E07 environment_integrity repository_snapshot=UNVERIFIED")
	} else {
		comparison := e07CompareIntegrity(beforeRepository, afterRepository)
		repositoryPass = comparison.Tree && comparison.Env && comparison.Tracked && comparison.GitState
		t.Logf("E07 environment_integrity %s", comparison.Summary())
	}
	if err := os.RemoveAll(state.tempRoot); err != nil {
		t.Log("E07 isolated validation cleanup=FAIL")
		return "FAIL"
	}
	if codexPass && privatePass && repositoryPass {
		return "PASS"
	}
	return "FAIL"
}
