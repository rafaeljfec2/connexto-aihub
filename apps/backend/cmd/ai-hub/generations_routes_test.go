package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/health"
	"aihub/internal/observability"
	"aihub/internal/operational"
	"aihub/internal/providers"
	"aihub/internal/records"
)

func TestGenerationsListReturnsRetainedRowsWithUnavailableEnrichment(t *testing.T) {
	router, executionID := newGenerationsTestRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/generations?limit=10", nil)
	request.Host = "localhost"
	request.Header.Set("Authorization", "Bearer test-gateway-key-1234567890-1234567890")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /v1/generations status = %d, want 200 (body %s)", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}

	var body struct {
		Generations []map[string]json.RawMessage `json:"generations"`
		Count       int                          `json:"count"`
		NextCursor  *string                      `json:"next_cursor"`
		Degraded    bool                         `json:"degraded"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode generation list: %v", err)
	}
	if body.Count != 1 || len(body.Generations) != 1 {
		t.Fatalf("generation page has count=%d rows=%d, want one retained row", body.Count, len(body.Generations))
	}
	var gotID string
	if err := json.Unmarshal(body.Generations[0]["execution_id"], &gotID); err != nil || gotID != executionID {
		t.Fatalf("generation execution_id = %q, err %v; want %q", gotID, err, executionID)
	}
	var usage struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body.Generations[0]["usage"], &usage); err != nil || usage.Status != "unavailable" {
		t.Fatalf("generation usage status = %q, err %v; want unavailable in file mode", usage.Status, err)
	}
	for _, forbidden := range []string{"prompt", "response_body", "api_key", "provider_session_id"} {
		if _, exists := body.Generations[0][forbidden]; exists {
			t.Errorf("generation list exposes forbidden field %q", forbidden)
		}
	}
}

func TestGenerationsRoutesRequireGatewayAuthorization(t *testing.T) {
	router, _ := newGenerationsTestRouter(t)
	for _, path := range []string{
		"/v1/generations",
		"/v1/generations/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Host = "localhost"
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Errorf("unauthenticated GET %s status = %d, want 401", path, response.Code)
		}
	}
}

func TestGenerationDetailRejectsUnknownExecution(t *testing.T) {
	router, _ := newGenerationsTestRouter(t)
	request := httptest.NewRequest(http.MethodGet,
		"/v1/generations/ffffffffffffffffffffffffffffffff", nil)
	request.Host = "localhost"
	request.Header.Set("Authorization", "Bearer test-gateway-key-1234567890-1234567890")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown generation detail status = %d, want 404", response.Code)
	}
	var body struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error.Type != "not_found" {
		t.Fatalf("unknown generation detail error type = %q, err %v; want not_found", body.Error.Type, err)
	}
}

func newGenerationsTestRouter(t *testing.T) (http.Handler, string) {
	t.Helper()

	logger := observability.NewLogger(io.Discard, slog.LevelInfo, "text")
	cfg := config.Default()
	cfg.Gateway.APIKey = "test-gateway-key-1234567890-1234567890"
	operatorGate, err := cfg.BuildOperatorGate()
	if err != nil {
		t.Fatalf("build operator gate: %v", err)
	}

	recordsModule := records.New(records.Options{
		Dir:               t.TempDir(),
		Logger:            logger,
		MaxAge:            cfg.Records.MaxAge,
		MaxCount:          cfg.Records.MaxCount,
		MaxStorageBytes:   cfg.Records.MaxStorageBytes,
		SegmentBytes:      cfg.Records.SegmentBytes,
		RetentionInterval: cfg.Records.RetentionInterval,
	})
	if err := recordsModule.Start(t.Context()); err != nil {
		t.Fatalf("start execution records: %v", err)
	}
	t.Cleanup(func() { _ = recordsModule.Stop(t.Context()) })

	executionID := strings.Repeat("a", 32)
	startedAt := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	recordsModule.OnExecutionFinished(execution.FinishedExecution{
		Result: execution.Result{
			ExecutionID: executionID,
			Outcome:     execution.OutcomeCompleted,
			Phase:       execution.PhaseForwarding,
			Duration:    25 * time.Millisecond,
		},
		RequestID: "req-test-123",
		Target:    execution.Target{ProviderID: "opencode-go", ModelID: "vendor/model"},
		StartedAt: startedAt,
	})
	waitForGenerationRecord(t, recordsModule, executionID)

	registry := providers.NewRegistry()
	providerManager := providers.NewRegistryManager(logger, registry)
	readiness := health.NewRegistry()
	eventHub := operational.NewHub(logger)
	router, err := buildGatewayRouter(cfg, logger, registry, providerManager, nil, nil, nil, recordsModule,
		readiness, nil, nil, nil, nil, nil, nil, nil, eventHub, operatorGate, nil)
	if err != nil {
		t.Fatalf("build gateway router: %v", err)
	}

	return router, executionID
}

func waitForGenerationRecord(t *testing.T, service *records.Service, executionID string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		page, err := service.List(records.Filter{Limit: 10})
		if err == nil && len(page.Records) == 1 && page.Records[0].ExecutionID == executionID {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("execution record %s was not indexed", executionID)
}
