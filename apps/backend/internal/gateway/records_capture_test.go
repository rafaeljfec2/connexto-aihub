package gateway_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/records"
)

func TestResponsesStreamStaysVerbatimWithRecordsObserver(t *testing.T) {
	logger, logBuffer := newTestLogger()
	recordsDir := t.TempDir()
	service := records.New(records.Options{
		Dir:               recordsDir,
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
	handler := gateway.NewResponsesHandlerWithManager(client, manager, logger, time.Second, nil, nil, nil, nil)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if capture.Body != expectedOpenCodeUpstreamBody(t, codexStylePayload) {
		t.Error("provider body differs from the original request")
	}
	if body != strings.Join(events, "") {
		t.Error("streamed response differs with record capture enabled")
	}
	if count := strings.Count(logBuffer.String(), "model_id=deepseek-v4.1-flash"); count != 2 {
		t.Errorf("model_id log entries = %d, want start and finish entries", count)
	}

	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("records Stop() returned error: %v", err)
	}
	recovered := records.New(records.Options{
		Dir:               recordsDir,
		Logger:            logger,
		MaxAge:            720 * time.Hour,
		MaxCount:          10000,
		MaxStorageBytes:   64 << 20,
		SegmentBytes:      1 << 20,
		RetentionInterval: time.Hour,
	})
	if err := recovered.Start(context.Background()); err != nil {
		t.Fatalf("recovered records Start() returned error: %v", err)
	}
	t.Cleanup(func() { _ = recovered.Stop(context.Background()) })

	request := httptest.NewRequest(http.MethodGet,
		"/v1/records/executions?model_id=deepseek-v4.1-flash", nil)
	responseRecorder := httptest.NewRecorder()
	gateway.NewExecutionRecordsHandler(recovered).ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("records query status = %d, want 200", responseRecorder.Code)
	}
	var listed struct {
		Records []struct {
			ModelID *string `json:"model_id"`
		} `json:"records"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode records query response: %v", err)
	}
	if len(listed.Records) != 1 || listed.Records[0].ModelID == nil ||
		*listed.Records[0].ModelID != "deepseek-v4.1-flash" {
		t.Fatal("recovered records query did not return the requested model_id")
	}
}

func TestResponsesHandlerContinuesWhenModelMetadataCannotBeExtracted(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "missing model", payload: `{"input":"hello"}`},
		{name: "empty model", payload: `{"model":"","input":"hello"}`},
		{name: "whitespace model", payload: `{"model":"   ","input":"hello"}`},
		{name: "incompatible model type", payload: `{"model":{"name":"unexpected"},"input":"hello"}`},
		{name: "invalid JSON", payload: `{"model":"deepseek-v4.1-flash"`},
		{name: "model exceeds record bound", payload: `{"model":"` + strings.Repeat("m", 4097) + `"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := newTestLogger()
			var capture upstreamRequest
			upstream := sseUpstream(t, &capture,
				[]string{"event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n"},
				0, nil)
			client := newEnabledClient(t, upstream.URL)
			observer := &executionTargetObserver{}
			manager := execution.NewManager(logger, providers.OpenCodeGoID,
				func(ctx context.Context, target execution.Target, payload []byte, contentType, sessionID string) (*http.Response, error) {
					return client.OpenTargetResponses(ctx, providers.ResponsesTarget{
						ProviderID: target.ProviderID,
						ModelID:    target.ModelID,
					}, payload, contentType, sessionID)
				}, time.Second, execution.WithObserver(observer))
			handler := gateway.NewResponsesHandlerWithManager(client, manager, logger, time.Second, nil, nil, nil, nil)

			response, _ := postResponses(t,
				servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
				test.payload, nil)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
			}
			wantBody := test.payload
			if parsed, err := gateway.OpenCodeResponsesPayloadForUpstream([]byte(test.payload)); err == nil {
				wantBody = string(parsed)
			}
			if capture.Body != wantBody {
				t.Error("provider body differs from the original request")
			}
			target, observed := observer.target()
			if !observed {
				t.Fatal("execution target was not observed")
			}
			if target.ModelID != "" {
				t.Errorf("observed model_id = %q, want empty", target.ModelID)
			}
		})
	}
}

type executionTargetObserver struct {
	mu     sync.Mutex
	value  execution.Target
	called bool
}

func (o *executionTargetObserver) OnExecutionFinished(finished execution.FinishedExecution) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.value = finished.Target
	o.called = true
}

func (o *executionTargetObserver) target() (execution.Target, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.value, o.called
}
