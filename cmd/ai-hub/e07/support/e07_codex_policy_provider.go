package support

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

type E07CodexPolicyProvider struct {
	Mu                      sync.Mutex
	Server                  *httptest.Server
	RequestCount            int
	GatewayAuthObserved     bool
	ToolOutputObserved      bool
	GatewayChildPresent     bool
	GatewayChildAbsent      bool
	ProviderChildPresent    bool
	ProviderChildAbsent     bool
	ProviderEnvChildPresent bool
	ProviderEnvChildAbsent  bool
	HubGatewayChildPresent  bool
	HubGatewayChildAbsent   bool
}

type E07CodexPolicySnapshot struct {
	RequestCount            int
	GatewayAuthObserved     bool
	ToolOutputObserved      bool
	GatewayChildPresent     bool
	GatewayChildAbsent      bool
	ProviderChildPresent    bool
	ProviderChildAbsent     bool
	ProviderEnvChildPresent bool
	ProviderEnvChildAbsent  bool
	HubGatewayChildPresent  bool
	HubGatewayChildAbsent   bool
}

type E07CodexPolicyRequest struct {
	Input []struct {
		Type   string          `json:"type"`
		Output json.RawMessage `json:"output"`
	} `json:"input"`
}

type E07CodexPolicyCommand struct {
	Command string `json:"cmd"`
	Workdir string `json:"workdir"`
	Login   bool   `json:"login"`
}

type E07CodexPolicyUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type E07CodexPolicyResponse struct {
	ID    string               `json:"id"`
	Usage *E07CodexPolicyUsage `json:"usage,omitempty"`
}

type E07CodexPolicyContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type E07CodexPolicyItem struct {
	Type      string                  `json:"type"`
	CallID    string                  `json:"call_id,omitempty"`
	Name      string                  `json:"name,omitempty"`
	Arguments string                  `json:"arguments,omitempty"`
	Role      string                  `json:"role,omitempty"`
	ID        string                  `json:"id,omitempty"`
	Content   []E07CodexPolicyContent `json:"content,omitempty"`
}

type E07CodexPolicyEvent struct {
	Type     string                  `json:"type"`
	Response *E07CodexPolicyResponse `json:"response,omitempty"`
	Item     *E07CodexPolicyItem     `json:"item,omitempty"`
}

func E07StartCodexPolicyProvider(gatewayKey, command, workdir string) *E07CodexPolicyProvider {
	provider := &E07CodexPolicyProvider{}
	provider.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		provider.Mu.Lock()
		provider.RequestCount++
		provider.GatewayAuthObserved = r.Header.Get("Authorization") == "Bearer "+gatewayKey
		provider.Mu.Unlock()
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid synthetic request", http.StatusBadRequest)
			return
		}
		var request E07CodexPolicyRequest
		if err := json.Unmarshal(body, &request); err != nil {
			http.Error(w, "invalid synthetic request", http.StatusBadRequest)
			return
		}
		for _, item := range request.Input {
			if item.Type == "function_call_output" {
				provider.ObserveToolOutput(item.Output)
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if provider.Snapshot().ToolOutputObserved {
			_, _ = io.WriteString(w, E07CodexPolicyCompletionSSE("resp-final", "Environment canary completed."))
			return
		}
		arguments, err := json.Marshal(E07CodexPolicyCommand{Command: command, Workdir: workdir})
		if err != nil {
			http.Error(w, "synthetic tool request could not be encoded", http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, E07CodexPolicyToolCallSSE(string(arguments)))
	}))
	return provider
}

func (provider *E07CodexPolicyProvider) ObserveToolOutput(output json.RawMessage) {
	provider.Mu.Lock()
	defer provider.Mu.Unlock()
	provider.ToolOutputObserved = true
	provider.GatewayChildPresent = bytes.Contains(output, []byte("AIHUB_CLIENT_KEY=PRESENT"))
	provider.GatewayChildAbsent = bytes.Contains(output, []byte("AIHUB_CLIENT_KEY=ABSENT"))
	provider.ProviderChildPresent = bytes.Contains(output, []byte("OPENCODE_API_KEY=PRESENT"))
	provider.ProviderChildAbsent = bytes.Contains(output, []byte("OPENCODE_API_KEY=ABSENT"))
	provider.ProviderEnvChildPresent = bytes.Contains(output, []byte(config.EnvProviderOpenCodeGoAPIKey+"=PRESENT"))
	provider.ProviderEnvChildAbsent = bytes.Contains(output, []byte(config.EnvProviderOpenCodeGoAPIKey+"=ABSENT"))
	provider.HubGatewayChildPresent = bytes.Contains(output, []byte("AIHUB_GATEWAY_API_KEY=PRESENT"))
	provider.HubGatewayChildAbsent = bytes.Contains(output, []byte("AIHUB_GATEWAY_API_KEY=ABSENT"))
}

func (provider *E07CodexPolicyProvider) Snapshot() E07CodexPolicySnapshot {
	provider.Mu.Lock()
	defer provider.Mu.Unlock()
	return E07CodexPolicySnapshot{
		RequestCount: provider.RequestCount, GatewayAuthObserved: provider.GatewayAuthObserved,
		ToolOutputObserved: provider.ToolOutputObserved, GatewayChildPresent: provider.GatewayChildPresent,
		GatewayChildAbsent: provider.GatewayChildAbsent, ProviderChildPresent: provider.ProviderChildPresent,
		ProviderChildAbsent: provider.ProviderChildAbsent, ProviderEnvChildPresent: provider.ProviderEnvChildPresent,
		ProviderEnvChildAbsent: provider.ProviderEnvChildAbsent, HubGatewayChildPresent: provider.HubGatewayChildPresent,
		HubGatewayChildAbsent: provider.HubGatewayChildAbsent,
	}
}

func E07CodexPolicyToolCallSSE(arguments string) string {
	return E07CodexPolicySSE(
		E07CodexPolicyEvent{Type: "response.created", Response: &E07CodexPolicyResponse{ID: "resp-tool"}},
		E07CodexPolicyEvent{Type: "response.output_item.done", Item: &E07CodexPolicyItem{
			Type: "function_call", CallID: "call-env", Name: "exec_command", Arguments: arguments,
		}},
		E07CodexPolicyEvent{Type: "response.completed", Response: &E07CodexPolicyResponse{ID: "resp-tool", Usage: &E07CodexPolicyUsage{}}},
	)
}

func E07CodexPolicyCompletionSSE(id, message string) string {
	return E07CodexPolicySSE(
		E07CodexPolicyEvent{Type: "response.output_item.done", Item: &E07CodexPolicyItem{
			Type: "message", Role: "assistant", ID: "msg-final",
			Content: []E07CodexPolicyContent{{Type: "output_text", Text: message}},
		}},
		E07CodexPolicyEvent{Type: "response.completed", Response: &E07CodexPolicyResponse{ID: id, Usage: &E07CodexPolicyUsage{}}},
	)
}

func E07CodexPolicySSE(events ...E07CodexPolicyEvent) string {
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
