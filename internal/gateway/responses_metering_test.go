package gateway_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/metering"
	"aihub/internal/providers"
)

type usageCaptureRepository struct {
	records         []metering.Record
	err             error
	contextCanceled bool
	hasDeadline     bool
}

func (r *usageCaptureRepository) Insert(ctx context.Context, record metering.Record) (bool, error) {
	r.contextCanceled = ctx.Err() != nil
	_, r.hasDeadline = ctx.Deadline()
	if r.err != nil {
		return false, r.err
	}
	r.records = append(r.records, record)
	return true, nil
}

type cancelOnTerminalWriter struct {
	*httptest.ResponseRecorder
	cancel context.CancelFunc
}

func (w *cancelOnTerminalWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseRecorder.Write(data)
	if bytes.Contains(data, []byte("response.completed")) {
		w.cancel()
	}
	return n, err
}

func (w *cancelOnTerminalWriter) Flush() {}

func meteringResponsesHandler(t *testing.T, upstreamURL string, repository metering.Repository) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := newEnabledClient(t, upstreamURL)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, client)
	manager := execution.NewMultiProviderManager(logger, func(
		ctx context.Context,
		target execution.Target,
		payload []byte,
		contentType string,
		sessionID string,
	) (*http.Response, error) {
		return registry.OpenTargetResponses(ctx, providers.ResponsesTarget{
			ProviderID: target.ProviderID,
			ModelID:    target.ModelID,
		}, payload, contentType, sessionID)
	}, time.Minute)
	usage := metering.NewService(repository, metering.WithLogger(logger))
	return gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, nil, nil, nil, usage)
}

func meteringUpstream(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func meteringRequest(t *testing.T) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"usage-model","input":"hello"}`))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func TestResponsesPersistsUsageAfterClientContextCancellationOnce(t *testing.T) {
	body := "event: response.completed\ndata: {\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}\n\n" +
		"event: response.completed\ndata: {\"usage\":{\"input_tokens\":4,\"output_tokens\":2,\"total_tokens\":6}}\n\n"
	upstream := meteringUpstream(t, http.StatusOK, body)
	repository := &usageCaptureRepository{}
	handler := meteringResponsesHandler(t, upstream.URL, repository)
	requestContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := meteringRequest(t).WithContext(requestContext)
	writer := &cancelOnTerminalWriter{ResponseRecorder: httptest.NewRecorder(), cancel: cancel}

	handler.ServeHTTP(writer, request)

	if requestContext.Err() == nil {
		t.Fatal("request context was not canceled by terminal response write")
	}
	if repository.contextCanceled || !repository.hasDeadline {
		t.Fatalf("persistence context canceled=%v deadline=%v", repository.contextCanceled, repository.hasDeadline)
	}
	if len(repository.records) != 1 {
		t.Fatalf("persisted records = %d, want one", len(repository.records))
	}
	if repository.records[0].InputTokens == nil || *repository.records[0].InputTokens != 4 {
		t.Fatalf("persisted usage = %+v, want final terminal usage", repository.records[0])
	}
}

func TestResponsesPersistsTerminalUsageFromProviderError(t *testing.T) {
	body := "event: response.completed\ndata: {\"usage\":{\"input_tokens\":3,\"output_tokens\":2,\"total_tokens\":5}}\n\n"
	upstream := meteringUpstream(t, http.StatusBadGateway, body)
	repository := &usageCaptureRepository{}
	handler := meteringResponsesHandler(t, upstream.URL, repository)
	writer := httptest.NewRecorder()
	handler.ServeHTTP(writer, meteringRequest(t))

	if writer.Code != http.StatusBadGateway || writer.Body.String() != body {
		t.Fatalf("response status=%d body=%q", writer.Code, writer.Body.String())
	}
	if len(repository.records) != 1 || repository.records[0].InputTokens == nil || *repository.records[0].InputTokens != 3 {
		t.Fatalf("persisted records = %+v", repository.records)
	}
}

func TestResponsesKeepsValidResponseWhenUsagePersistenceFails(t *testing.T) {
	body := "event: response.completed\ndata: {\"usage\":{\"input_tokens\":3,\"output_tokens\":2,\"total_tokens\":5}}\n\n"
	upstream := meteringUpstream(t, http.StatusOK, body)
	repository := &usageCaptureRepository{err: errors.New("storage unavailable")}
	handler := meteringResponsesHandler(t, upstream.URL, repository)
	writer := httptest.NewRecorder()
	handler.ServeHTTP(writer, meteringRequest(t))

	if writer.Code != http.StatusOK || writer.Body.String() != body {
		t.Fatalf("response status=%d body=%q", writer.Code, writer.Body.String())
	}
	if len(repository.records) != 0 {
		t.Fatalf("persisted records = %d after repository failure", len(repository.records))
	}
}
