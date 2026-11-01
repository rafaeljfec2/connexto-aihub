package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE07CodexResumeContinuesFromTemporaryWorkspace(t *testing.T) {
	if os.Getenv(e07CanaryEnvVar) != "1" {
		t.Skip("opt-in Codex CLI resume canary is disabled")
	}
	codexPath, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal("Codex CLI is unavailable while the resume canary is enabled")
	}

	tempRoot := t.TempDir()
	codexHome := filepath.Join(tempRoot, "codex-home")
	if err := e07EnsureCodexRuntimeDirectories(codexHome); err != nil {
		t.Fatal("temporary Codex runtime directories could not be prepared")
	}
	const gatewayKey = "e07-gateway-synthetic-resume"
	parentEnvironment := []string{
		"PATH=" + os.Getenv("PATH"),
		"LANG=C.UTF-8",
		"HOME=" + filepath.Join(tempRoot, "parent-home"),
	}
	codexEnvironment := e07CodexEnvironment(parentEnvironment, codexHome, gatewayKey)
	versionCommand := exec.Command(codexPath, "--version")
	versionCommand.Env = codexEnvironment
	versionOutput, err := versionCommand.Output()
	if err != nil || strings.TrimSpace(string(versionOutput)) != e07CodexVersion {
		t.Fatal("Codex CLI does not match the recorded E07 validation release")
	}
	provider := e07StartCodexPolicyProvider(gatewayKey, "python3 -c 'print(\"READY\")'", tempRoot)
	defer provider.Server.Close()
	if err := e07WriteCodexConfig(codexHome, provider.Server.URL, codexEnvironment); err != nil {
		t.Fatal("temporary Codex configuration could not be written")
	}

	first, err := runE07CodexWithParentEnvironment(codexPath, codexHome, gatewayKey,
		parentEnvironment, tempRoot, "Start a synthetic conversation.",
		"exec", "--json", "--sandbox", "workspace-write", "--skip-git-repo-check", "-")
	if err != nil || first.ExitCode != 0 || first.ThreadID == "" {
		t.Fatal("Codex could not create a local synthetic conversation")
	}
	firstTurnRequests := provider.snapshot().requestCount
	if firstTurnRequests != 2 {
		t.Fatalf("initial local Codex turn made %d Responses requests, want 2", firstTurnRequests)
	}

	resumed, err := runE07CodexWithParentEnvironment(codexPath, codexHome, gatewayKey,
		parentEnvironment, tempRoot, "Continue the synthetic conversation.",
		"exec", "resume", "--json", "--skip-git-repo-check", first.ThreadID, "-")
	if err != nil || resumed.ExitCode != 0 {
		t.Fatalf("Codex could not resume the local synthetic conversation: exit_code=%d", resumed.ExitCode)
	}
	if got := provider.snapshot().requestCount; got != firstTurnRequests+1 {
		t.Fatalf("resumed Codex turn made %d total Responses requests, want %d", got, firstTurnRequests+1)
	}
}
