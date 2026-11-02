package codex

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"aihub/internal/config"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestE07CodexShellPolicyDisablesUserProfile(t *testing.T) {
	policy := e07support.E07CodexShellPolicyTOML([]string{
		"HOME=/tmp/e07-codex-home",
		"PATH=/usr/bin",
		"TMPDIR=/tmp/e07-codex-home/tmp",
	})
	if !strings.Contains(policy, "experimental_use_profile = false") {
		t.Fatal("Codex shell environment policy does not disable user profile loading")
	}
}

func TestE07StableCodexPreflightRejectsActiveProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the active Codex process probe currently targets Linux")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal("test executable could not be resolved")
	}
	command := exec.Command(executable, "-test.run", "^TestE07CodexProcessNameHelper$")
	command.Env = []string{"E07_CODEX_PROCESS_HELPER=1"}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal("synthetic process output pipe could not be created")
	}
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal("synthetic process input pipe could not be created")
	}
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		t.Fatal("synthetic Codex process could not be started")
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		_ = command.Wait()
	})
	ready := []byte{0}
	if _, err := io.ReadFull(stdout, ready); err != nil {
		t.Fatal("synthetic Codex process did not announce readiness")
	}
	active, err := e07support.E07CodexProcessActive()
	if err != nil || !active {
		t.Fatal("synthetic Codex process was not detected")
	}

	_, err = e07support.E07RequireStableCodexTree()
	if err == nil || !strings.Contains(err.Error(), "active Codex process") {
		t.Fatal("Codex preflight did not reject an active process before inspecting global state")
	}
}

func TestE07CodexEnvironmentExcludesProviderCredentials(t *testing.T) {
	const gatewayKey = "e07-gateway-synthetic-environment-test"
	const providerKey = "e07-provider-synthetic-environment-test"
	parentEnvironment := []string{
		"PATH=/usr/bin",
		"LANG=C.UTF-8",
		"HOME=/tmp/e07-parent-home",
		"AIHUB_GATEWAY_API_KEY=" + gatewayKey,
		"OPENCODE_API_KEY=" + providerKey,
		config.EnvProviderOpenCodeGoAPIKey + "=" + providerKey,
	}
	codexEnvironment := e07support.E07CodexEnvironment(parentEnvironment, "/tmp/e07-codex-home", gatewayKey)
	if e07support.EnvironmentMap(codexEnvironment)[e07support.E07ClientKeyEnv] != gatewayKey {
		t.Fatal("Codex process did not receive the synthetic gateway key")
	}
	for _, name := range []string{"AIHUB_GATEWAY_API_KEY", "OPENCODE_API_KEY", config.EnvProviderOpenCodeGoAPIKey} {
		if e07support.E07EnvironmentHasKey(codexEnvironment, name) {
			t.Fatalf("Codex process environment contains a forbidden credential variable: %s", name)
		}
	}

	shellEnvironment, err := e07support.E07CodexShellEnvironment(codexEnvironment)
	if err != nil {
		t.Fatal("Codex shell environment could not be constructed")
	}
	for _, name := range []string{e07support.E07ClientKeyEnv, "AIHUB_GATEWAY_API_KEY", "OPENCODE_API_KEY", config.EnvProviderOpenCodeGoAPIKey} {
		if e07support.E07EnvironmentHasKey(shellEnvironment, name) {
			t.Fatalf("Codex shell environment contains a forbidden credential variable: %s", name)
		}
	}
}

func TestE07CodexShellEnvironmentPolicyIsEffective(t *testing.T) {
	if os.Getenv(e07support.E07CanaryEnvVar) != "1" {
		t.Skip("opt-in Codex shell canary is disabled; set " + e07support.E07CanaryEnvVar + "=1 to run it")
	}
	if runtime.GOOS != "linux" {
		t.Skip("the isolated shell canary currently targets Linux")
	}
	codexPath, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal("Codex CLI is unavailable while the opt-in canary is enabled")
	}

	tempRoot := t.TempDir()
	codexHome := filepath.Join(tempRoot, "codex-home")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal("temporary Codex home could not be created")
	}
	if err := e07support.E07EnsureCodexRuntimeDirectories(codexHome); err != nil {
		t.Fatal("temporary Codex runtime directories could not be prepared")
	}
	globalBefore, err := e07support.E07RequireStableCodexTree()
	if err != nil {
		t.Fatalf("global Codex tree is not stable; isolated Codex was not started: %s", err)
	}
	beforeManifest, err := e07support.E07SaveCodexTreeManifest(tempRoot, "before", globalBefore)
	if err != nil {
		t.Fatal("private global Codex metadata manifest could not be saved")
	}
	t.Cleanup(func() {
		globalAfter, snapshotErr := e07support.E07SnapshotCodexTree()
		if snapshotErr != nil {
			t.Error("global Codex metadata could not be rechecked after the probe")
			return
		}
		afterManifest, saveErr := e07support.E07SaveCodexTreeManifest(tempRoot, "after", globalAfter)
		beforeSaved, beforeErr := e07support.E07LoadCodexTreeManifest(beforeManifest)
		afterSaved, afterErr := e07support.E07LoadCodexTreeManifest(afterManifest)
		if saveErr != nil || beforeErr != nil || afterErr != nil || !e07support.E07CodexTreeUnchanged(beforeSaved, afterSaved) {
			t.Error("isolated Codex canary changed the global Codex tree")
		}
	})

	gatewayKey := "e07-gateway-synthetic-canary"
	providerKey := "e07-provider-synthetic-canary"
	parentEnvironment := []string{
		"PATH=" + os.Getenv("PATH"),
		"LANG=C.UTF-8",
		"HOME=" + filepath.Join(tempRoot, "parent-home"),
		"AIHUB_GATEWAY_API_KEY=" + gatewayKey,
		"OPENCODE_API_KEY=" + providerKey,
		config.EnvProviderOpenCodeGoAPIKey + "=" + providerKey,
	}
	codexEnvironment := e07support.E07CodexEnvironment(parentEnvironment, codexHome, gatewayKey)
	codexValues := e07support.EnvironmentMap(codexEnvironment)
	if codexValues[e07support.E07ClientKeyEnv] != gatewayKey ||
		e07support.E07EnvironmentHasKey(codexEnvironment, "OPENCODE_API_KEY") ||
		e07support.E07EnvironmentHasKey(codexEnvironment, config.EnvProviderOpenCodeGoAPIKey) ||
		e07support.E07EnvironmentHasKey(codexEnvironment, "AIHUB_GATEWAY_API_KEY") {
		t.Fatal("Codex process environment did not isolate the synthetic credentials")
	}
	for _, name := range []string{"HOME", "CODEX_HOME", "CODEX_SQLITE_HOME", "TMPDIR", "TEMP", "TMP", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR"} {
		if !strings.HasPrefix(codexValues[name], tempRoot+string(os.PathSeparator)) {
			t.Fatal("Codex process environment did not use temporary runtime paths")
		}
	}

	versionCommand := exec.Command(codexPath, "--version")
	versionCommand.Env = codexEnvironment
	versionOutput, err := versionCommand.Output()
	if err != nil {
		t.Fatal("isolated Codex CLI version probe failed")
	}
	version := strings.TrimSpace(string(versionOutput))
	t.Logf("E07 isolated_codex_version=%s", version)
	if version != e07support.E07CodexVersion {
		t.Fatalf("installed Codex CLI does not match the E07 validation target: %s", version)
	}

	toolCommand := "python3 -c 'import os; print(\"AIHUB_CLIENT_KEY=\" + (\"PRESENT\" if os.getenv(\"AIHUB_CLIENT_KEY\") else \"ABSENT\")); print(\"OPENCODE_API_KEY=\" + (\"PRESENT\" if os.getenv(\"OPENCODE_API_KEY\") else \"ABSENT\")); print(\"AIHUB_PROVIDER_OPENCODEGO_API_KEY=\" + (\"PRESENT\" if os.getenv(\"AIHUB_PROVIDER_OPENCODEGO_API_KEY\") else \"ABSENT\")); print(\"AIHUB_GATEWAY_API_KEY=\" + (\"PRESENT\" if os.getenv(\"AIHUB_GATEWAY_API_KEY\") else \"ABSENT\"))'"
	provider := e07support.E07StartCodexPolicyProvider(gatewayKey, toolCommand, tempRoot)
	defer provider.Server.Close()
	if err := e07support.E07WriteCodexConfig(codexHome, provider.Server.URL, codexEnvironment); err != nil {
		t.Fatal("temporary Codex configuration could not be written")
	}
	configBytes, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatal("temporary Codex configuration could not be inspected")
	}
	if bytes.Contains(configBytes, []byte(gatewayKey)) || bytes.Contains(configBytes, []byte(providerKey)) {
		t.Fatal("temporary Codex configuration contains a synthetic credential value")
	}
	if !bytes.Contains(configBytes, []byte(`sandbox_mode = "workspace-write"`)) ||
		bytes.Contains(configBytes, []byte("approval_policy =")) {
		t.Fatal("temporary Codex configuration did not preserve normal sandbox and approval controls")
	}

	run, err := e07support.RunE07CodexWithParentEnvironment(codexPath, codexHome, gatewayKey, parentEnvironment, tempRoot,
		"Use exec_command once to report only whether the gateway and provider canary variables are present.",
		"exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
	if err != nil || !run.Ran || run.ExitCode != 0 {
		t.Fatal("Codex could not complete the local synthetic environment canary")
	}
	state := provider.Snapshot()
	if !state.GatewayAuthObserved {
		t.Fatal("Codex did not authenticate to the local Responses endpoint with its configured gateway key")
	}
	if !state.ToolOutputObserved || !state.GatewayChildAbsent || !state.ProviderChildAbsent ||
		!state.ProviderEnvChildAbsent || !state.HubGatewayChildAbsent {
		t.Fatal("local Responses endpoint did not observe the synthetic shell environment result")
	}
	if state.GatewayChildPresent || state.ProviderChildPresent || state.ProviderEnvChildPresent || state.HubGatewayChildPresent {
		t.Fatal("Codex shell tool inherited a synthetic credential")
	}
	if state.RequestCount != 2 {
		t.Fatal("local Responses canary received an unexpected request count")
	}
}
