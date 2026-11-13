package gateway_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/catalog"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/routing"
)

func newToolsPassthroughHandler(t *testing.T, openCodeURL, openRouterURL string) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	openCode, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: openCodeURL, APIKey: openCodeTestKey}, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	openRouter, err := providers.NewOpenRouterClient(providers.Settings{BaseURL: openRouterURL + "/api/v1", APIKey: openRouterTestKey, Enabled: true}, nil)
	if err != nil {
		t.Fatalf("NewOpenRouterClient() error = %v", err)
	}
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.OpenRouterID, openRouter)

	allowlistIDs := []string{
		"glm-5.3-flash",
		"deepseek-v4.1-flash",
		"claude-haiku-5-5",
		"gpt-6-luna",
		"anthropic/claude-haiku-4.5",
	}
	opencodeModels := make([]providers.ModelDescriptor, 0, 4)
	openRouterModels := make([]providers.ModelDescriptor, 0, 1)
	for _, id := range allowlistIDs {
		desc := gatewayTestDescriptor(id)
		if id == "anthropic/claude-haiku-4.5" {
			openRouterModels = append(openRouterModels, desc)
			continue
		}
		opencodeModels = append(opencodeModels, desc)
	}

	now := time.Now()
	catalogManager := gatewayTestCatalog(t, now, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return opencodeModels, nil
		},
		providers.OpenRouterID: func(context.Context) ([]providers.ModelDescriptor, error) {
			return openRouterModels, nil
		},
	})

	manager := execution.NewMultiProviderManager(logger,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{
				ProviderID: target.ProviderID,
				ModelID:    target.ModelID,
			}, payload, contentType, sessionID)
		}, time.Minute)

	return gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, &gateway.ResponsesRouting{
		Catalog: catalogManager,
		Statuses: func() []providers.Status {
			return []providers.Status{
				{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateHealthy},
				{ID: providers.OpenRouterID, Enabled: true, State: providers.StateHealthy},
			}
		},
		StaleAfter: 24 * time.Hour,
	}, nil, nil, nil)
}

func TestAutoAndAllowlistModelsForwardToolPayloadUnmodified(t *testing.T) {
	toolFixture := `{"instructions":"stay intact","tool_choice":"auto","tools":[{"type":"custom","name":"probe_tool"}],"input":[{"type":"input_text","text":"invoke probe_tool once"}]}`

	cases := []struct {
		name    string
		model   string
		headers map[string]string
	}{
		{name: "virtual auto", model: "aihub/auto"},
		{name: "openrouter allowlist", model: "anthropic/claude-haiku-4.5", headers: map[string]string{"X-AIHub-Provider": providers.OpenRouterID}},
		{name: "opencode glm", model: "glm-5.3-flash"},
		{name: "opencode deepseek", model: "deepseek-v4.1-flash"},
		{name: "opencode claude", model: "claude-haiku-5-5"},
		{name: "opencode luna", model: "gpt-6-luna"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var openCodeCaptured string
			var openRouterCaptured string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				openCodeCaptured = string(body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"id":"upstream-ok","output":[{"type":"function_call","name":"probe_tool","call_id":"call_probe","arguments":"{}"}]}`)
			}))
			defer upstream.Close()

			openRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				openRouterCaptured = string(body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"id":"upstream-ok","output":[{"type":"function_call","name":"probe_tool","call_id":"call_probe","arguments":"{}"}]}`)
			}))
			defer openRouter.Close()

			handler := newToolsPassthroughHandler(t, upstream.URL, openRouter.URL)
			payload := `{"model":"` + tc.model + `",` + strings.TrimPrefix(toolFixture, "{")
			req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}

			var captured string
			switch {
			case tc.headers != nil && tc.headers["X-AIHub-Provider"] == providers.OpenRouterID:
				captured = openRouterCaptured
			case tc.model == "anthropic/claude-haiku-4.5":
				captured = openRouterCaptured
				if captured == "" {
					captured = openCodeCaptured
				}
			case tc.model == routing.VirtualModelAuto:
				captured = openCodeCaptured
				if captured == "" {
					captured = openRouterCaptured
				}
			default:
				captured = openCodeCaptured
			}

			if strings.Contains(captured, "SYSTEM_DIRECTIVE") || strings.Contains(captured, "RESPONSES_TOOL_SYSTEM_DIRECTIVE") {
				t.Fatalf("upstream payload was rewritten: %s", captured)
			}
			if !strings.Contains(captured, `"instructions":"stay intact"`) {
				t.Fatalf("instructions missing: %s", captured)
			}
			if !strings.Contains(captured, `"tool_choice":"auto"`) {
				t.Fatalf("tool_choice missing: %s", captured)
			}
			if !strings.Contains(captured, `"type":"custom"`) || !strings.Contains(captured, "probe_tool") {
				t.Fatalf("custom tools missing: %s", captured)
			}

			var forwarded map[string]json.RawMessage
			if err := json.Unmarshal([]byte(captured), &forwarded); err != nil {
				t.Fatalf("upstream json: %v", err)
			}
			if tc.model == routing.VirtualModelAuto {
				var modelID string
				if err := json.Unmarshal(forwarded["model"], &modelID); err != nil || modelID == "" || modelID == routing.VirtualModelAuto {
					t.Fatalf("auto must substitute concrete model, got %q body=%s", modelID, captured)
				}
				if !routing.IsAutoAllowlistTarget(routing.Target{ProviderID: providers.OpenCodeGoID, ModelID: modelID}) &&
					!routing.IsAutoAllowlistTarget(routing.Target{ProviderID: providers.OpenRouterID, ModelID: modelID}) {
					t.Fatalf("resolved model %q outside allowlist", modelID)
				}
			} else if !strings.Contains(captured, tc.model) {
				t.Fatalf("expected model %q in body: %s", tc.model, captured)
			}
		})
	}
}

func TestAutoToolResponseIncludesFunctionCallFromUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"tool-resp","output":[{"type":"function_call","name":"probe_tool","call_id":"call_1","arguments":"{}"}]}`)
	}))
	defer upstream.Close()
	openRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	defer openRouter.Close()

	logger, _ := newTestLogger()
	handler := newOpenCodeRoutingHandler(t, upstream.URL, logger, time.Minute, "gpt-6-luna")
	payload := `{"model":"aihub/auto","tool_choice":"required","tools":[{"type":"function","name":"probe_tool","parameters":{"type":"object"}}],"input":"call probe_tool"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "function_call") && !strings.Contains(rec.Body.String(), "probe_tool") {
		t.Fatalf("expected tool call in client response, got %s", rec.Body.String())
	}
}
