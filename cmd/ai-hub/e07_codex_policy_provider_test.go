package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"aihub/internal/config"
)

type e07CodexPolicyProvider struct {
	mu                      sync.Mutex
	Server                  *httptest.Server
	requestCount            int
	gatewayAuthObserved     bool
	toolOutputObserved      bool
	gatewayChildPresent     bool
	gatewayChildAbsent      bool
	providerChildPresent    bool
	providerChildAbsent     bool
	providerEnvChildPresent bool
	providerEnvChildAbsent  bool
	hubGatewayChildPresent  bool
	hubGatewayChildAbsent   bool
}

type e07CodexPolicySnapshot struct {
	requestCount            int
	gatewayAuthObserved     bool
	toolOutputObserved      bool
	gatewayChildPresent     bool
	gatewayChildAbsent      bool
	providerChildPresent    bool
	providerChildAbsent     bool
	providerEnvChildPresent bool
	providerEnvChildAbsent  bool
	hubGatewayChildPresent  bool
	hubGatewayChildAbsent   bool
}

type e07CodexPolicyRequest struct {
	Input []struct {
		Type   string          `json:"type"`
		Output json.RawMessage `json:"output"`
	} `json:"input"`
}

type e07CodexPolicyCommand struct {
	Command string `json:"cmd"`
	Workdir string `json:"workdir"`
	Login   bool   `json:"login"`
}

type e07CodexPolicyUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type e07CodexPolicyResponse struct {
	ID    string               `json:"id"`
	Usage *e07CodexPolicyUsage `json:"usage,omitempty"`
}

type e07CodexPolicyContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type e07CodexPolicyItem struct {
	Type      string                  `json:"type"`
	CallID    string                  `json:"call_id,omitempty"`
	Name      string                  `json:"name,omitempty"`
	Arguments string                  `json:"arguments,omitempty"`
	Role      string                  `json:"role,omitempty"`
	ID        string                  `json:"id,omitempty"`
	Content   []e07CodexPolicyContent `json:"content,omitempty"`
}

type e07CodexPolicyEvent struct {
	Type     string                  `json:"type"`
	Response *e07CodexPolicyResponse `json:"response,omitempty"`
	Item     *e07CodexPolicyItem     `json:"item,omitempty"`
}

func e07StartCodexPolicyProvider(gatewayKey, command, workdir string) *e07CodexPolicyProvider {
	provider := &e07CodexPolicyProvider{}
	provider.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		provider.mu.Lock()
		provider.requestCount++
		provider.gatewayAuthObserved = r.Header.Get("Authorization") == "Bearer "+gatewayKey
		provider.mu.Unlock()
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid synthetic request", http.StatusBadRequest)
			return
		}
		var request e07CodexPolicyRequest
		if err := json.Unmarshal(body, &request); err != nil {
			http.Error(w, "invalid synthetic request", http.StatusBadRequest)
			return
		}
		for _, item := range request.Input {
			if item.Type == "function_call_output" {
				provider.observeToolOutput(item.Output)
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if provider.snapshot().toolOutputObserved {
			_, _ = io.WriteString(w, e07CodexPolicyCompletionSSE("resp-final", "Environment canary completed."))
			return
		}
		arguments, err := json.Marshal(e07CodexPolicyCommand{Command: command, Workdir: workdir})
		if err != nil {
			http.Error(w, "synthetic tool request could not be encoded", http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, e07CodexPolicyToolCallSSE(string(arguments)))
	}))
	return provider
}

func (provider *e07CodexPolicyProvider) observeToolOutput(output json.RawMessage) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.toolOutputObserved = true
	provider.gatewayChildPresent = bytes.Contains(output, []byte("AIHUB_CLIENT_KEY=PRESENT"))
	provider.gatewayChildAbsent = bytes.Contains(output, []byte("AIHUB_CLIENT_KEY=ABSENT"))
	provider.providerChildPresent = bytes.Contains(output, []byte("OPENCODE_API_KEY=PRESENT"))
	provider.providerChildAbsent = bytes.Contains(output, []byte("OPENCODE_API_KEY=ABSENT"))
	provider.providerEnvChildPresent = bytes.Contains(output, []byte(config.EnvProviderOpenCodeGoAPIKey+"=PRESENT"))
	provider.providerEnvChildAbsent = bytes.Contains(output, []byte(config.EnvProviderOpenCodeGoAPIKey+"=ABSENT"))
	provider.hubGatewayChildPresent = bytes.Contains(output, []byte("AIHUB_GATEWAY_API_KEY=PRESENT"))
	provider.hubGatewayChildAbsent = bytes.Contains(output, []byte("AIHUB_GATEWAY_API_KEY=ABSENT"))
}

func (provider *e07CodexPolicyProvider) snapshot() e07CodexPolicySnapshot {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return e07CodexPolicySnapshot{
		requestCount: provider.requestCount, gatewayAuthObserved: provider.gatewayAuthObserved,
		toolOutputObserved: provider.toolOutputObserved, gatewayChildPresent: provider.gatewayChildPresent,
		gatewayChildAbsent: provider.gatewayChildAbsent, providerChildPresent: provider.providerChildPresent,
		providerChildAbsent: provider.providerChildAbsent, providerEnvChildPresent: provider.providerEnvChildPresent,
		providerEnvChildAbsent: provider.providerEnvChildAbsent, hubGatewayChildPresent: provider.hubGatewayChildPresent,
		hubGatewayChildAbsent: provider.hubGatewayChildAbsent,
	}
}

func e07CodexPolicyToolCallSSE(arguments string) string {
	return e07CodexPolicySSE(
		e07CodexPolicyEvent{Type: "response.created", Response: &e07CodexPolicyResponse{ID: "resp-tool"}},
		e07CodexPolicyEvent{Type: "response.output_item.done", Item: &e07CodexPolicyItem{
			Type: "function_call", CallID: "call-env", Name: "exec_command", Arguments: arguments,
		}},
		e07CodexPolicyEvent{Type: "response.completed", Response: &e07CodexPolicyResponse{ID: "resp-tool", Usage: &e07CodexPolicyUsage{}}},
	)
}

func e07CodexPolicyCompletionSSE(id, message string) string {
	return e07CodexPolicySSE(
		e07CodexPolicyEvent{Type: "response.output_item.done", Item: &e07CodexPolicyItem{
			Type: "message", Role: "assistant", ID: "msg-final",
			Content: []e07CodexPolicyContent{{Type: "output_text", Text: message}},
		}},
		e07CodexPolicyEvent{Type: "response.completed", Response: &e07CodexPolicyResponse{ID: id, Usage: &e07CodexPolicyUsage{}}},
	)
}

func e07CodexPolicySSE(events ...e07CodexPolicyEvent) string {
	var stream strings.Builder
	for _, event := range events {
		encoded, err := json.Marshal(event)
		if err != nil {
			panic("synthetic Codex policy event could not be encoded")
		}
		_, _ = fmt.Fprintf(&stream, "event: %s\ndata: %s\n\n", event.Type, encoded)
	}
	return stream.String()
}
