package main

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
	"strconv"
	"strings"
	"time"
)

const (
	e07CodexVersion            = "codex-cli 0.164.0-alpha.11.1"
	e07ModelID                 = "deepseek-v4.1-flash"
	e07ClientKeyEnv            = "AIHUB_CLIENT_KEY"
	e07ReplayMarker            = "E07-SYNTHETIC-MARKER-7F2C"
	e07MaxCodexOutputBytes     = 8 << 20
	e07MaxCodexDiagnosticBytes = 1 << 20
)

type e07CodexRun struct {
	Ran              bool
	ThreadID         string
	Message          string
	ProgressiveSeen  bool
	ExitCode         int
	EnvironmentLimit string
}

type e07CodexEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Item     struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"item"`
}

func e07WriteCodexConfig(home, baseURL string) error {
	if err := os.MkdirAll(home, 0o700); err != nil {
		return errors.New("isolated Codex home could not be created")
	}
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
`, strconv.Quote(e07ModelID), strconv.Quote(baseURL+"/v1"), e07ClientKeyEnv)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		return errors.New("isolated Codex configuration could not be written")
	}
	return nil
}

func runE07Codex(codexPath, codexHome, gatewayKey, workdir, prompt string, args ...string) (e07CodexRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, codexPath, args...)
	command.Dir = workdir
	command.Env = e07CodexEnvironment(os.Environ(), codexHome, gatewayKey)
	command.Stdin = strings.NewReader(prompt)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return e07CodexRun{}, errors.New("Codex output could not be captured")
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return e07CodexRun{}, errors.New("Codex diagnostics could not be captured")
	}
	if err := command.Start(); err != nil {
		return e07CodexRun{}, errors.New("Codex process could not be started")
	}
	stderrDone := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(io.LimitReader(stderr, e07MaxCodexDiagnosticBytes+1))
		if len(data) > e07MaxCodexDiagnosticBytes {
			data = data[:e07MaxCodexDiagnosticBytes]
		}
		_, _ = io.Copy(io.Discard, stderr)
		stderrDone <- data
	}()

	var captured bytes.Buffer
	outputExceeded := false
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 32*1024), 4*1024*1024)
	for scanner.Scan() {
		if captured.Len()+len(scanner.Bytes())+1 > e07MaxCodexOutputBytes {
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
		return e07CodexRun{}, errors.New("Codex output exceeded the validation bound")
	}
	if waitErr != nil {
		result := e07CodexRun{Ran: true, ExitCode: 1}
		if exitErr := new(exec.ExitError); errors.As(waitErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
		result.EnvironmentLimit = e07CodexEnvironmentLimit(stderrOutput)
		return result, errors.New("Codex invocation failed")
	}
	return parseE07CodexEvents(captured.Bytes()), nil
}

func e07CodexEnvironmentLimit(diagnostics []byte) string {
	text := strings.ToLower(string(diagnostics))
	for _, marker := range []string{"bwrap", "landlock", "approval required", "operation not permitted", "sandbox initialization failed", "sandbox is unavailable"} {
		if strings.Contains(text, marker) {
			return "sandbox_or_approval"
		}
	}
	return ""
}

func parseE07CodexEvents(output []byte) e07CodexRun {
	result := e07CodexRun{Ran: true, ExitCode: 0}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(make([]byte, 32*1024), 4*1024*1024)
	for scanner.Scan() {
		var event e07CodexEvent
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

func e07EnvironmentHasKey(environment []string, name string) bool {
	for _, value := range environment {
		key, _, found := strings.Cut(value, "=")
		if found && key == name {
			return true
		}
	}
	return false
}
