package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"aihub/internal/config"
)

type e07LiveSetup struct {
	Config       config.Config
	GatewayKey   string
	ProviderKey  string
	CodexPath    string
	CodexHome    string
	DataDir      string
	BasicDir     string
	ReplayDir    string
	ToolDir      string
	TempRoot     string
	Revision     string
	CodexVersion string
	GoVersion    string
	OSClass      string
}

func e07PrepareLiveSetup(t *testing.T) *e07LiveSetup {
	t.Helper()
	root, err := e07RepositoryRoot("")
	if err != nil {
		t.Fatal("repository root could not be resolved")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal("user home directory could not be resolved")
	}
	tempRoot, err := os.MkdirTemp("", "aihub-e07-")
	if err != nil {
		t.Fatal("isolated validation state could not be created")
	}
	if err := os.Chmod(tempRoot, 0o700); err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("isolated validation state could not be protected")
	}
	beforeEnvironment := e07EnvironmentSnapshot()
	beforeCodex, err := e07FingerprintTree(filepath.Join(home, ".codex"))
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatalf("global Codex integrity baseline could not be captured: %s", err)
	}
	beforeRepository, err := e07SnapshotRepository(root)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("repository integrity baseline could not be captured")
	}
	manifestDirectory, err := e07PrivateManifestDirectory(root, tempRoot)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private integrity manifest directory could not be resolved")
	}
	manifestPath, err := e07SaveIntegrityManifest(manifestDirectory, beforeRepository)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private integrity manifest could not be written")
	}
	codexHome := filepath.Join(tempRoot, "codex-home")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("isolated CODEX_HOME could not be created")
	}
	cleanup := func() {
		e07RestoreEnvironment(beforeEnvironment)
		afterCodex, codexErr := e07FingerprintTree(filepath.Join(home, ".codex"))
		codexResult := e07IntegrityResult(codexErr == nil && beforeCodex == afterCodex)
		if codexResult != "PASS" {
			t.Error("environment integrity: global Codex tree changed")
		}
		beforeManifest, manifestErr := e07LoadIntegrityManifest(manifestPath)
		afterRepository, repositoryErr := e07SnapshotRepository(root)
		if manifestErr != nil || repositoryErr != nil {
			t.Error("environment integrity: repository snapshot could not be verified")
			t.Logf("E07 environment_integrity codex_tree=%s repository_snapshot=UNVERIFIED", codexResult)
		} else {
			comparison := e07CompareIntegrity(beforeManifest, afterRepository)
			if !comparison.Tree {
				t.Error("environment integrity: repository tree changed during validation")
			}
			if !comparison.Env {
				t.Error("environment integrity: project environment files changed")
			}
			if !comparison.Tracked {
				t.Error("environment integrity: tracked file content changed during validation")
			}
			if !comparison.GitState {
				t.Error("environment integrity: Git state changed during validation")
			}
			t.Logf("E07 environment_integrity codex_tree=%s %s", codexResult, comparison.Summary())
		}
		if err := os.RemoveAll(tempRoot); err != nil {
			t.Error("isolated validation state could not be removed")
		}
	}
	t.Cleanup(cleanup)
	if err := config.LoadDotEnv(filepath.Join(root, ".env")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal("provider environment could not be loaded")
	}
	providerKey := os.Getenv(config.EnvProviderOpenCodeGoAPIKey)
	if providerKey == "" {
		providerKey = os.Getenv("OPENCODE_API_KEY")
	}
	if providerKey == "" {
		t.Skip("NOT RUN — provider credential is unavailable")
	}
	if os.Getenv(config.EnvProviderOpenCodeGoAPIKey) == "" {
		if err := os.Setenv(config.EnvProviderOpenCodeGoAPIKey, providerKey); err != nil {
			t.Fatal("provider credential could not be assigned to the Hub environment")
		}
	}
	matchesBaseline, err := e07ProductionBaselineMatches(root, e07BaselineRevision)
	if err != nil {
		t.Fatal("production baseline could not be verified")
	}
	if !matchesBaseline {
		t.Fatal("validation is not running on the E06 production baseline")
	}
	revision, err := e07GitRevision(root)
	if err != nil {
		t.Fatal("Git revision could not be captured")
	}
	gatewayKey := os.Getenv(config.EnvGatewayAPIKey)
	if gatewayKey == "" {
		gatewayKey, err = e07TemporaryGatewayKey(providerKey)
	}
	if err != nil || os.Setenv(config.EnvGatewayAPIKey, gatewayKey) != nil {
		t.Fatal("temporary gateway credential could not be prepared")
	}
	codexPath, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("NOT RUN — Codex CLI is unavailable")
	}
	versionCommand := exec.Command(codexPath, "--version")
	versionCommand.Env = e07CodexEnvironment(os.Environ(), codexHome, gatewayKey)
	versionOutput, err := versionCommand.Output()
	if err != nil {
		t.Skip("NOT RUN — Codex CLI version could not be read")
	}
	codexVersion := strings.TrimSpace(string(versionOutput))
	if codexVersion != e07CodexVersion {
		t.Skip("NOT RUN — Codex CLI release differs from the recorded validation target")
	}
	cfg := config.Default()
	cfg.Gateway.APIKey = gatewayKey
	cfg.Provider.OpenCodeGo.APIKey = providerKey
	if value := os.Getenv(config.EnvProviderOpenCodeGoBaseURL); value != "" {
		cfg.Provider.OpenCodeGo.BaseURL = value
	}
	setup := &e07LiveSetup{
		Config: cfg, GatewayKey: gatewayKey, ProviderKey: providerKey,
		CodexPath: codexPath, CodexHome: codexHome,
		DataDir: filepath.Join(tempRoot, "hub-data"), BasicDir: filepath.Join(tempRoot, "basic-workspace"),
		ReplayDir: filepath.Join(tempRoot, "replay-workspace"), ToolDir: filepath.Join(tempRoot, "tool-workspace"),
		TempRoot: tempRoot, Revision: revision, CodexVersion: codexVersion,
		GoVersion: runtime.Version(), OSClass: runtime.GOOS + "/" + runtime.GOARCH,
	}
	if err := os.Setenv(config.EnvDataDir, setup.DataDir); err != nil {
		t.Fatal("dedicated AI Hub data directory could not be configured")
	}
	for _, directory := range []string{setup.CodexHome, setup.DataDir, setup.BasicDir, setup.ReplayDir, setup.ToolDir} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			_ = os.RemoveAll(tempRoot)
			t.Fatal("isolated validation directory could not be created")
		}
	}
	setup.Config.Catalog.DataDir = setup.DataDir
	return setup
}

func e07EnvironmentSnapshot() map[string]string {
	values := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			values[key] = value
		}
	}
	return values
}

func e07RestoreEnvironment(before map[string]string) {
	for key := range e07EnvironmentSnapshot() {
		if _, existed := before[key]; !existed {
			_ = os.Unsetenv(key)
		}
	}
	for key, value := range before {
		_ = os.Setenv(key, value)
	}
}
