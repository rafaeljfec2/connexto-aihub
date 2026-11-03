package gateway_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

const providerCredential = "oc_sk_gateway_fixture_credential"

// codexStylePayload reproduces the request shape validated in the spike,
// including fields the gateway must never rewrite.
const codexStylePayload = `{"model":"deepseek-v4.1-flash","instructions":"be brief",` +
	`"input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}],` +
	`"tools":[{"type":"function","name":"read_file"},{"type":"custom","name":"apply_patch"}],` +
	`"tool_choice":"auto","parallel_tool_calls":true,"reasoning":{"effort":"high"},` +
	`"store":false,"stream":true,"include":["reasoning.encrypted_content"],` +
	`"prompt_cache_key":"pc_1","text":{"verbosity":"low"},"service_tier":"auto",` +
	`"client_metadata":{"agent":"codex"}}`

type upstreamRequest struct {
	Method        string
	Path          string
	Authorization string
	Session       string
	ContentType   string
	Body          string
}

// sseUpstream records the request and streams scripted events with real
// flushing so the passthrough copy path is exercised.
func sseUpstream(t *testing.T, capture *upstreamRequest, events []string, gap time.Duration, closeEvents chan struct{}) *httptest.Server {
	t.Helper()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		*capture = upstreamRequest{
			Method:        r.Method,
			Path:          r.URL.Path,
			Authorization: r.Header.Get("Authorization"),
			Session:       r.Header.Get("X-Opencode-Session"),
			ContentType:   r.Header.Get("Content-Type"),
			Body:          string(body),
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Errorf("streaming writer must support flush")
			return
		}

		for _, event := range events {
			if _, err := fmt.Fprint(w, event); err != nil {
				return
			}
			flusher.Flush()
			select {
			case <-time.After(gap):
			case <-r.Context().Done():
				return
			case <-closeEvents:
				return
			}
		}
	}))
	t.Cleanup(upstream.Close)

	return upstream
}

func newEnabledClient(t *testing.T, baseURL string) *providers.Client {
	t.Helper()

	client, err := providers.NewClient(config.ProviderOpenCodeGo{
		BaseURL: baseURL,
		APIKey:  providerCredential,
	}, nil)
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}

	return client
}

func postResponses(t *testing.T, target string, body string, headers map[string]string) (*http.Response, string) {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, target, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build requests: %v", err)
	}

	for name, value := range headers {
		request.Header.Set(name, value)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST %s: %v", target, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	return response, string(content)
}

// servedThroughRouter builds a real listener with a deliberately tiny write
// timeout so streaming tests prove the inactivity deadline re-arming.
func servedThroughRouter(t *testing.T, handler http.Handler, writeTimeout time.Duration) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	httpServer := &http.Server{
		Handler:           handler,
		WriteTimeout:      writeTimeout,
		ReadHeaderTimeout: time.Second,
	}

	go func() { _ = httpServer.Serve(listener) }()
	t.Cleanup(func() { _ = httpServer.Close() })

	return "http://" + listener.Addr().String()
}

func TestResponsesHandlerRejectsProviderWithoutCredential(t *testing.T) {
	logger, _ := newTestLogger()

	disabled, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: "https://example.com/go"}, nil)
	if err != nil {
		t.Fatalf("NewClient() returned error: %v", err)
	}

	handler := gateway.NewResponsesHandler(disabled, logger, time.Minute, nil)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.StatusCode)
	}

	if !strings.Contains(body, "provider_unavailable") {
		t.Fatalf("body %q must mention provider_unavailable", body)
	}

	if strings.Contains(body, providerCredential) {
		t.Fatalf("body leaks the credential")
	}
}

func TestResponsesHandlerPassesRequestAndStreamVerbatim(t *testing.T) {
	logger, logBuffer := newTestLogger()

	var capture upstreamRequest

	events := []string{
		"event: response.created\ndata: {\"type\":\"response.created\"}\n\n",
		"event: response.output_item.added\ndata: {\"type\":\"reasoning\"}\n\n",
		"event: response.output_text.delta\ndata: {\"delta\":\"hello\"}\n\n",
		"event: response.completed\ndata: {\"usage\":{\"input_tokens\":1,\"opencode\":{\"billing_source\":\"go\"}}}\n\n",
	}

	upstream := sseUpstream(t, &capture, events, 20*time.Millisecond, nil)
	handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, 2*time.Second, nil)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	if got, want := response.Header.Get("Content-Type"), "text/event-stream"; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}

	assertions := []struct {
		name, got, want string
	}{
		{"method", capture.Method, http.MethodPost},
		{"path", capture.Path, "/openai/v1/responses"},
		{"authorization", capture.Authorization, "Bearer " + providerCredential},
		{"content type", capture.ContentType, "application/json"},
		{"body", capture.Body, codexStylePayload},
	}
	for _, item := range assertions {
		if item.got != item.want {
			t.Errorf("upstream %s = %q, want %q", item.name, item.got, item.want)
		}
	}

	if capture.Session == "" {
		t.Fatalf("upstream must receive a session header")
	}

	if !strings.HasPrefix(capture.Session, "aihub-ses-") {
		t.Fatalf("generated session %q must use the aihub-ses- prefix", capture.Session)
	}

	if body != strings.Join(events, "") {
		t.Errorf("streamed payload changed in transit:\n got: %q\nwant: %q", body, strings.Join(events, ""))
	}

	if logBuffer.contains(providerCredential) {
		t.Fatalf("logs leaked the credential: %q", logBuffer.String())
	}
}

func TestResponsesHandlerPreservesSuppliedSession(t *testing.T) {
	var capture upstreamRequest

	upstream := sseUpstream(t, &capture, []string{
		"event: response.completed\ndata: {}\n\n",
	}, 0, nil)
	logger, _ := newTestLogger()
	handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, time.Minute, nil)

	_, _ = postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, map[string]string{"X-Opencode-Session": "ses_client_supplied"})

	if capture.Session != "ses_client_supplied" {
		t.Fatalf("session header = %q, want the supplied value", capture.Session)
	}
}

func TestResponsesHandlerSessionIsIndependentFromCorrelationID(t *testing.T) {
	var capture upstreamRequest

	upstream := sseUpstream(t, &capture, []string{"event: response.completed\ndata: {}\n\n"}, 0, nil)

	logger, _ := newTestLogger()
	handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, time.Minute, nil)
	target := servedThroughRouter(t, handler, time.Hour) + "/v1/responses"

	request, err := http.NewRequest(http.MethodPost, target, strings.NewReader(codexStylePayload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	request.Header.Set("X-Request-Id", "corr_123")
	request.Header.Set("X-Opencode-Session", "ses_context_1")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, response.Body)

	if capture.Session != "ses_context_1" {
		t.Fatalf("session header = %q, want ses_context_1", capture.Session)
	}
}

func TestResponsesHandlerPreservesUpstreamErrorEnvelope(t *testing.T) {
	const errorBody = `{"error":{"type":"server_error","message":"Upstream request failed: Invalid credential"}}`

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(errorBody))
	}))
	defer upstream.Close()

	logger, logBuffer := newTestLogger()
	handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, time.Minute, nil)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}

	if body != errorBody {
		t.Fatalf("body = %q, want the upstream error envelope", body)
	}

	if logBuffer.contains(providerCredential) {
		t.Fatalf("logs leaked the credential")
	}
}

func TestResponsesHandlerAnswersSafe502WhenUpstreamUnreachable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	dead := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	logger, _ := newTestLogger()
	handler := gateway.NewResponsesHandler(newEnabledClient(t, "http://"+dead), logger, time.Minute, nil)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, time.Hour)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", response.StatusCode)
	}

	if strings.Contains(body, providerCredential) || strings.Contains(body, dead) {
		t.Fatalf("502 body must be safe, got %q", body)
	}
}

// Streaming timeout policy.

func TestStreamSurvivesBeyondOrdinaryWriteTimeout(t *testing.T) {
	var capture upstreamRequest

	gap := 30 * time.Millisecond
	events := make([]string, 0, 12)
	for index := range 12 {
		events = append(events, fmt.Sprintf("event: response.output_text.delta\ndata: {\"i\":%d}\n\n", index))
	}

	upstream := sseUpstream(t, &capture, events, gap, nil)
	logger, logBuffer := newTestLogger()
	handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, time.Second, nil)

	response, body := postResponses(t,
		servedThroughRouter(t, handler, 80*time.Millisecond)+"/v1/responses",
		codexStylePayload, nil)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	want := strings.Join(events, "")
	if body != want {
		t.Fatalf("stream truncated by the ordinary write timeout:\n got %q\nwant %q", body, want)
	}

	if logBuffer.contains("inactivity") {
		t.Fatalf("no inactivity abort expected while chunks flow, log: %q", logBuffer.String())
	}
}

func TestStreamAbortsAfterInactivityWithoutChunks(t *testing.T) {
	var capture upstreamRequest

	events := []string{"event: response.created\ndata: {}\n\n"}
	upstream := sseUpstream(t, &capture, events, time.Second, nil) // long silence after first event
	logger, logBuffer := newTestLogger()
	handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, 100*time.Millisecond, nil)
	target := servedThroughRouter(t, handler, time.Hour) + "/v1/responses"

	done := make(chan struct{})
	go func() {
		defer close(done)

		response, err := http.Post(target, "application/json", strings.NewReader(codexStylePayload))
		if err != nil {
			return
		}

		defer func() { _ = response.Body.Close() }()
		_, _ = io.ReadAll(response.Body)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("handler did not abort the stream after the inactivity window")
	}

	if !logBuffer.contains("inactivity") {
		t.Fatalf("expected an inactivity abort log, got %q", logBuffer.String())
	}
}

func TestClientDisconnectAbortsUpstreamStream(t *testing.T) {
	var capture upstreamRequest

	events := make([]string, 0, 40)
	for range 40 {
		events = append(events, "event: response.output_text.delta\ndata: {}\n\n")
	}

	upstream := sseUpstream(t, &capture, events, 25*time.Millisecond, nil)
	logger, _ := newTestLogger()
	handler := gateway.NewResponsesHandler(newEnabledClient(t, upstream.URL), logger, time.Minute, nil)
	target := servedThroughRouter(t, handler, time.Hour) + "/v1/responses"

	request, err := http.NewRequest(http.MethodPost, target, strings.NewReader(codexStylePayload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request = request.WithContext(ctx)

	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return
		}

		defer func() { _ = response.Body.Close() }()
		_, _ = io.Copy(io.Discard, response.Body)
	}()

	// Give the stream room to start, then disconnect the client mid-stream.
	select {
	case <-time.After(200 * time.Millisecond):
	case <-clientDone:
		t.Fatalf("client finished before cancellation was requested")
	}

	cancel()

	select {
	case <-clientDone:
	case <-time.After(5 * time.Second):
		t.Fatalf("handler did not abort after client disconnection")
	}
}
