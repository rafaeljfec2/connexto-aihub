package support

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

type E07LiveSetup struct {
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
	Observer     *E07ProviderObserver
	Integrity    *E07LiveIntegrityGuard
}

func E07PrepareLiveSetup(t *testing.T) *E07LiveSetup {
	t.Helper()
	if err := E07RequirePrivateCodexMount(os.Getenv(E07PrivateRootEnv)); err != nil {
		t.Fatal("live E2E requires a read-only hidden host Codex mount and private runtime paths")
	}
	root, err := E07RepositoryRoot("")
	if err != nil {
		t.Fatal("repository root could not be resolved")
	}
	tempRoot, err := os.MkdirTemp("", "aihub-e07-")
	if err != nil {
		t.Fatal("isolated validation state could not be created")
	}
	if err := os.Chmod(tempRoot, 0o700); err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("isolated validation state could not be protected")
	}
	beforeEnvironment := E07EnvironmentSnapshot()
	beforeCodex, err := E07SnapshotCodexTree()
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private Codex state baseline could not be captured")
	}
	manifestDirectory, err := E07PrivateManifestDirectory(root, tempRoot)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private integrity manifest directory could not be resolved")
	}
	if err := os.MkdirAll(manifestDirectory, 0o700); err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private integrity manifest directory could not be created")
	}
	beforeCodexManifest, err := E07SaveCodexTreeManifest(manifestDirectory, "before", beforeCodex)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private global Codex metadata manifest could not be saved")
	}
	beforeRepository, err := E07SnapshotRepository(root)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("repository integrity baseline could not be captured")
	}
	manifestPath, err := E07SaveIntegrityManifest(manifestDirectory, beforeRepository)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private integrity manifest could not be written")
	}
	codexHome := filepath.Join(tempRoot, "codex-home")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("isolated CODEX_HOME could not be created")
	}
	if err := E07EnsureCodexRuntimeDirectories(codexHome); err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("isolated Codex runtime directories could not be prepared")
	}
	privateRoot := os.Getenv(E07PrivateRootEnv)
	if !E07PathWithinRoot(privateRoot, tempRoot) || E07PathWithinRoot(root, tempRoot) {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("validation state is outside the private runtime mount")
	}
	for _, directory := range E07CodexRuntimeDirectories(codexHome) {
		if !E07PathWithinRoot(privateRoot, directory) {
			_ = os.RemoveAll(tempRoot)
			t.Fatal("Codex state is outside the private runtime mount")
		}
	}
	beforePrivateCodex, err := E07SnapshotPrivateCodexState(codexHome)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private Codex state baseline could not be captured")
	}
	beforePrivateManifest, err := E07SaveCodexTreeManifest(manifestDirectory, "private-before", beforePrivateCodex)
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		t.Fatal("private Codex state baseline could not be saved")
	}
	var setup *E07LiveSetup
	integrityState := E07LiveIntegrityState{
		Root: root, TempRoot: tempRoot, ManifestDirectory: manifestDirectory,
		BeforeEnvironment: beforeEnvironment, BeforeCodexManifest: beforeCodexManifest,
		BeforePrivateManifest: beforePrivateManifest, PrivateCodexHome: codexHome,
		ManifestPath: manifestPath,
	}
	cleanup := func() {
		var observer *E07ProviderObserver
		if setup != nil {
			observer = setup.Observer
		}
		E07RunLiveCleanupScenario(t, observer, "credential_isolation_repository_integrity", func() string {
			return E07FinalizeLiveIntegrity(t, integrityState)
		})
	}
	t.Cleanup(cleanup)
	environmentHadProviderKey := E07EnvironmentHasKey(os.Environ(), config.EnvProviderOpenCodeGoAPIKey)
	if err := config.LoadDotEnv(filepath.Join(root, ".env")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal("provider environment could not be loaded")
	}
	providerKey := E07ProviderCredential(os.Environ())
	if providerKey == "" {
		t.Skip("NOT RUN — provider credential is unavailable")
	}
	providerKeyOrigin := "dotenv"
	if environmentHadProviderKey {
		providerKeyOrigin = "environment"
	}
	t.Logf("E07 provider_credential_origin=%s", providerKeyOrigin)
	matchesBaseline, err := E07ProductionBaselineMatches(root, E07BaselineRevision)
	if err != nil {
		t.Fatal("production baseline could not be verified")
	}
	if !matchesBaseline {
		t.Fatal("validation is not running on the E06 production baseline")
	}
	revision, err := E07GitRevision(root)
	if err != nil {
		t.Fatal("Git revision could not be captured")
	}
	gatewayKey := E07GatewayCredential(os.Environ())
	if gatewayKey == "" {
		t.Fatal("configured gateway credential is unavailable")
	}
	if err := os.Setenv(config.EnvGatewayAPIKey, gatewayKey); err != nil {
		t.Fatal("configured gateway credential could not be installed in the Hub environment")
	}
	codexPath, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("NOT RUN — Codex CLI is unavailable")
	}
	versionCommand := exec.Command(codexPath, "--version")
	versionCommand.Env = E07CodexEnvironment(os.Environ(), codexHome, gatewayKey)
	versionOutput, err := versionCommand.Output()
	if err != nil {
		t.Fatal("Codex CLI version could not be read before the bounded scenario")
	}
	codexVersion := strings.TrimSpace(string(versionOutput))
	if codexVersion != E07CodexVersion {
		t.Fatalf("Codex CLI release differs from the recorded validation target %q", E07CodexVersion)
	}
	cfg := config.Default()
	cfg.Gateway.APIKey = gatewayKey
	cfg.Provider.OpenCodeGo.APIKey = providerKey
	if value := os.Getenv(config.EnvProviderOpenCodeGoBaseURL); value != "" {
		cfg.Provider.OpenCodeGo.BaseURL = value
	}
	setup = &E07LiveSetup{
		Config: cfg, GatewayKey: gatewayKey, ProviderKey: providerKey,
		CodexPath: codexPath, CodexHome: codexHome,
		DataDir: filepath.Join(tempRoot, "hub-data"), BasicDir: filepath.Join(tempRoot, "basic-workspace"),
		ReplayDir: filepath.Join(tempRoot, "replay-workspace"), ToolDir: filepath.Join(tempRoot, "tool-workspace"),
		TempRoot: tempRoot, Revision: revision, CodexVersion: codexVersion,
		GoVersion: runtime.Version(), OSClass: runtime.GOOS + "/" + runtime.GOARCH,
		Integrity: &E07LiveIntegrityGuard{
			Root: root, PrivateCodexHome: codexHome, BaselineCodex: beforeCodex,
			BaselineRepository: beforeRepository,
		},
	}
	if err := os.Setenv(config.EnvDataDir, setup.DataDir); err != nil {
		t.Fatal("dedicated AI Hub data directory could not be configured")
	}
	for _, directory := range []string{setup.CodexHome, setup.DataDir, setup.BasicDir, setup.ReplayDir, setup.ToolDir} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal("isolated validation directory could not be created")
		}
	}
	setup.Config.Catalog.DataDir = setup.DataDir
	return setup
}

func E07EnvironmentSnapshot() map[string]string {
	values := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			values[key] = value
		}
	}
	return values
}

func E07RestoreEnvironment(before map[string]string) {
	for key := range E07EnvironmentSnapshot() {
		if _, existed := before[key]; !existed {
			_ = os.Unsetenv(key)
		}
	}
	for key, value := range before {
		_ = os.Setenv(key, value)
	}
}

// e07ProviderCredential resolves the provider credential from the only variable
// the AI Hub production configuration consumes. The historical OPENCODE_API_KEY
// name appears in local configuration but is never read by the runtime, so it
// must not be substituted for the production variable.
func E07ProviderCredential(environment []string) string {
	if !E07EnvironmentHasKey(environment, config.EnvProviderOpenCodeGoAPIKey) {
		return ""
	}

	return EnvironmentMap(environment)[config.EnvProviderOpenCodeGoAPIKey]
}
