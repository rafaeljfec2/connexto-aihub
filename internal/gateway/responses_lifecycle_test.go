package gateway_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

func TestResponsesHandlerRejectsExecutionAfterDrainGateCloses(t *testing.T) {
	var upstreamCalls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamCalls++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	client := newEnabledClient(t, upstream.URL)
	logger, _ := newTestLogger()
	manager := execution.NewManager(logger, providers.OpenCodeGoID,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return client.OpenTargetResponses(ctx, providers.ResponsesTarget{
				ProviderID: target.ProviderID,
				ModelID:    target.ModelID,
			}, payload, contentType, sessionID)
		}, time.Second)
	manager.BeginDrain()
	handler := gateway.NewResponsesHandlerWithManager(client, manager, logger, time.Second, nil)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body = %q", response.StatusCode, body)
	}
	if !strings.Contains(body, "execution_unavailable") {
		t.Fatalf("body = %q, want execution_unavailable", body)
	}
	if upstreamCalls != 0 {
		t.Fatalf("upstream calls = %d, want none", upstreamCalls)
	}
}
