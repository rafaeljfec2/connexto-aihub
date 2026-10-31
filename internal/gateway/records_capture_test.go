package gateway_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/records"
)

func TestResponsesStreamStaysVerbatimWithRecordsObserver(t *testing.T) {
	logger, _ := newTestLogger()
	service := records.New(records.Options{
		Dir:               t.TempDir(),
		Logger:            logger,
		MaxAge:            720 * time.Hour,
		MaxCount:          10000,
		MaxStorageBytes:   64 << 20,
		SegmentBytes:      1 << 20,
		RetentionInterval: time.Hour,
	})
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("records Start() returned error: %v", err)
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	events := []string{
		"event: response.created\ndata: {\"type\":\"response.created\"}\n\n",
		"event: response.output_text.delta\ndata: {\"delta\":\"hello\"}\n\n",
		"event: response.completed\ndata: {\"usage\":{\"input_tokens\":1}}\n\n",
	}

	var capture upstreamRequest
	upstream := sseUpstream(t, &capture, events, 10*time.Millisecond, nil)
	client := newEnabledClient(t, upstream.URL)
	manager := execution.NewManager(logger, providers.OpenCodeGoID,
		func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
			return client.OpenTargetResponses(ctx, providers.ResponsesTarget{
				ProviderID: target.ProviderID,
				ModelID:    target.ModelID,
			}, payload, contentType, sessionID)
		}, time.Second, execution.WithObserver(service))
	handler := gateway.NewResponsesHandlerWithManager(client, manager, logger, time.Second)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if capture.Body != codexStylePayload {
		t.Errorf("upstream body = %q, want %q", capture.Body, codexStylePayload)
	}
	if body != strings.Join(events, "") {
		t.Errorf("streamed payload changed with record capture enabled:\n got: %q\nwant: %q",
			body, strings.Join(events, ""))
	}

	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("records Stop() returned error: %v", err)
	}
	page, err := service.List(records.Filter{Limit: 10})
	if err != nil {
		t.Fatalf("records List() returned error: %v", err)
	}
	if len(page.Records) != 1 || page.Records[0].Outcome != records.OutcomeCompleted {
		t.Fatalf("retained records = %+v, want one completed execution", page.Records)
	}
}
