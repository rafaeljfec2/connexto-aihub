package support

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	E07CodexVersion            = "codex-cli 0.164.0-alpha.11.1"
	E07ModelID                 = "deepseek-v4.1-flash"
	E07ClientKeyEnv            = "AIHUB_CLIENT_KEY"
	E07ReplayMarker            = "E07-SYNTHETIC-MARKER-7F2C"
	E07MaxCodexOutputBytes     = 8 << 20
	E07MaxCodexDiagnosticBytes = 1 << 20
)

type E07CodexRun struct {
	Ran              bool
	ThreadID         string
	Message          string
	ProgressiveSeen  bool
	ExitCode         int
	EnvironmentLimit string
}

type E07CodexEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Item     struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"item"`
}

func E07WriteCodexConfig(home, baseURL string, processEnvironment []string) error {
	if err := os.MkdirAll(home, 0o700); err != nil {
		return errors.New("isolated Codex home could not be created")
	}
	shellEnvironment, err := E07CodexShellEnvironment(processEnvironment)
	if err != nil {
		return err
	}
	shellPolicy := E07CodexShellPolicyTOML(shellEnvironment)
	config := fmt.Sprintf(`model = %s
model_provider = "aihub"
sandbox_mode = "workspace-write"

[model_providers.aihub]
name = "AI Hub OpenCode Go"
base_url = %s
wire_api = "responses"
env_key = %q
request_max_retries = 0
stream_max_retries = 0
`, strconv.Quote(E07ModelID), strconv.Quote(baseURL+"/v1"), E07ClientKeyEnv) + shellPolicy
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		return errors.New("isolated Codex configuration could not be written")
	}
	return nil
}

func E07CodexShellPolicyTOML(environment []string) string {
	entries := append([]string(nil), environment...)
	sort.Strings(entries)
	quoted := make([]string, 0, len(entries))
	for _, entry := range entries {
		name, value, found := strings.Cut(entry, "=")
		if found {
			quoted = append(quoted, name+" = "+strconv.Quote(value))
		}
	}
	return "\n[shell_environment_policy]\ninherit = \"none\"\nignore_default_excludes = false\nset = { " +
		strings.Join(quoted, ", ") + " }\nexperimental_use_profile = false\n"
}

func RunE07Codex(codexPath, codexHome, gatewayKey, workdir, prompt string, args ...string) (E07CodexRun, error) {
	return RunE07CodexWithParentEnvironment(codexPath, codexHome, gatewayKey, os.Environ(), workdir, prompt, args...)
}

func RunE07CodexWithParentEnvironment(codexPath, codexHome, gatewayKey string, parentEnvironment []string, workdir, prompt string, args ...string) (E07CodexRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, codexPath, args...)
	command.Dir = workdir
	command.Env = E07CodexEnvironment(parentEnvironment, codexHome, gatewayKey)
	command.Stdin = strings.NewReader(prompt)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return E07CodexRun{}, errors.New("Codex output could not be captured")
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return E07CodexRun{}, errors.New("Codex diagnostics could not be captured")
	}
	if err := command.Start(); err != nil {
		return E07CodexRun{}, errors.New("Codex process could not be started")
	}
	stderrDone := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(io.LimitReader(stderr, E07MaxCodexDiagnosticBytes+1))
		if len(data) > E07MaxCodexDiagnosticBytes {
			data = data[:E07MaxCodexDiagnosticBytes]
		}
		_, _ = io.Copy(io.Discard, stderr)
		stderrDone <- data
	}()

	var captured bytes.Buffer
	outputExceeded := false
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 32*1024), 4*1024*1024)
	for scanner.Scan() {
		if captured.Len()+len(scanner.Bytes())+1 > E07MaxCodexOutputBytes {
			outputExceeded = true
			continue
		}
		_, _ = captured.Write(scanner.Bytes())
		_ = captured.WriteByte('\n')
	}
	scanErr := scanner.Err()
	if scanErr != nil {
		_, _ = io.Copy(io.Discard, stdout)
	}
	waitErr := command.Wait()
	stderrOutput := <-stderrDone
	if scanErr != nil || outputExceeded {
		return E07CodexRun{}, errors.New("Codex output exceeded the validation bound")
	}
	if waitErr != nil {
		result := E07CodexRun{Ran: true, ExitCode: 1}
		if exitErr := new(exec.ExitError); errors.As(waitErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
		result.EnvironmentLimit = E07CodexEnvironmentLimit(stderrOutput)
		return result, errors.New("Codex invocation failed")
	}
	return ParseE07CodexEvents(captured.Bytes()), nil
}

func E07CodexEnvironmentLimit(diagnostics []byte) string {
	text := strings.ToLower(string(diagnostics))
	for _, marker := range []string{"bwrap", "landlock", "approval required", "operation not permitted", "sandbox initialization failed", "sandbox is unavailable"} {
		if strings.Contains(text, marker) {
			return "sandbox_or_approval"
		}
	}
	return ""
}

func ParseE07CodexEvents(output []byte) E07CodexRun {
	result := E07CodexRun{Ran: true, ExitCode: 0}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(make([]byte, 32*1024), 4*1024*1024)
	for scanner.Scan() {
		var event E07CodexEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		if event.Type == "thread.started" && event.ThreadID != "" {
			result.ThreadID = event.ThreadID
		}
		if event.Type == "item.completed" && event.Item.Type == "agent_message" {
			result.Message += event.Item.Text
		}
		if strings.Contains(event.Type, "delta") {
			result.ProgressiveSeen = true
		}
	}
	return result
}

func E07EnvironmentHasKey(environment []string, name string) bool {
	for _, value := range environment {
		key, _, found := strings.Cut(value, "=")
		if found && key == name {
			return true
		}
	}
	return false
}
