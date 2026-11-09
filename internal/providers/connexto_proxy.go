package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"aihub/internal/config"
)

const (
	connextoProxyModels    = "/models"
	connextoProxyResponses = "/responses"
	connextoProxyHealth    = "/health"
	connextoProxyErrorBody = `{"error":{"type":"upstream_error","message":"Connexto Proxy request failed"}}`
)

// ConnextoProxyClient implements the OpenAI Responses API through Connexto Proxy CLI LLMs.
// Requests remain opaque so provider-specific model IDs and fields are kept intact.
type ConnextoProxyClient struct {
	mu                sync.RWMutex
	baseURL           string
	apiKey            string
	gatewayKey        string
	enabled           bool
	logicalProviderID string
	inferenceOptional bool
	httpClient        *http.Client
}

func logicalProviderLabel(id string) string {
	if id != "" {
		return id
	}
	return CursorID
}

// NewConnextoProxyClient builds an adapter with a bounded shared provider transport.
func NewConnextoProxyClient(settings Settings, httpClient *http.Client) (*ConnextoProxyClient, error) {
	if err := config.ValidateBaseURL(settings.BaseURL); err != nil {
		return nil, fmt.Errorf("provider %s: %w", logicalProviderLabel(settings.LogicalProviderID), err)
	}
	if httpClient == nil {
		httpClient = NewConnextoProxyHTTPClient()
	}
	logicalID := settings.LogicalProviderID
	if logicalID == "" {
		logicalID = CursorID
	}
	inferenceOptional := settings.InferenceOptional
	if logicalID == AntigravityID {
		inferenceOptional = true
	}
	return &ConnextoProxyClient{
		baseURL:           strings.TrimRight(settings.BaseURL, "/"),
		apiKey:            settings.APIKey,
		gatewayKey:        settings.ManagementKey,
		enabled:           settings.Enabled,
		logicalProviderID: logicalID,
		inferenceOptional: inferenceOptional,
		httpClient:        httpClient,
	}, nil
}

// Reconfigure atomically changes endpoint, credential, and enabled state.
func (c *ConnextoProxyClient) Reconfigure(settings Settings) error {
	if err := config.ValidateBaseURL(settings.BaseURL); err != nil {
		return fmt.Errorf("provider %s: %w", logicalProviderLabel(settings.LogicalProviderID), err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	logicalID := settings.LogicalProviderID
	if logicalID == "" {
		logicalID = CursorID
	}
	c.baseURL = strings.TrimRight(settings.BaseURL, "/")
	c.apiKey = settings.APIKey
	c.gatewayKey = settings.ManagementKey
	c.enabled = settings.Enabled
	if settings.LogicalProviderID != "" {
		c.logicalProviderID = settings.LogicalProviderID
	}
	if settings.LogicalProviderID == AntigravityID {
		c.inferenceOptional = true
	} else {
		c.inferenceOptional = settings.InferenceOptional
	}
	return nil
}

// Enabled is false when disabled. For Cursor, a non-empty inference API key is required.
// For Antigravity, the same proxy credential works as for Cursor: a non-empty API key
// enables the adapter. Alternatively (or in addition), gateway auth uses the stored
// management key or AIHUB_CONNEXTO_PROXY_GATEWAY_KEY.
func (c *ConnextoProxyClient) Enabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.enabled {
		return false
	}
	if c.logicalProviderID == AntigravityID {
		return strings.TrimSpace(c.apiKey) != "" || antigravityGatewayConfigured(c.gatewayKey)
	}
	return c.apiKey != ""
}

func antigravityGatewayConfigured(storedGateway string) bool {
	if strings.TrimSpace(storedGateway) != "" {
		return true
	}
	return AntigravityGatewayKeyFromEnvironment()
}

// AntigravityGatewayKeyFromEnvironment reports whether AIHUB_CONNEXTO_PROXY_GATEWAY_KEY is set.
func AntigravityGatewayKeyFromEnvironment() bool {
	return strings.TrimSpace(os.Getenv("AIHUB_CONNEXTO_PROXY_GATEWAY_KEY")) != ""
}

// ListModels retrieves Connexto Proxy's model list.
func (c *ConnextoProxyClient) ListModels(ctx context.Context) (*http.Response, error) {
	response, err := c.do(ctx, http.MethodGet, connextoProxyModels, nil, "")
	if err != nil {
		return nil, err
	}
	sanitizeConnextoProxyErrorResponse(response)
	return c.filterModelsListResponse(response)
}

// OpenTargetResponses forwards a stateless Responses request without rewriting it.
func (c *ConnextoProxyClient) OpenTargetResponses(ctx context.Context, target ResponsesTarget, payload []byte, contentType, _ string) (*http.Response, error) {
	c.mu.RLock()
	logicalID := c.logicalProviderID
	c.mu.RUnlock()
	if logicalID == "" {
		logicalID = CursorID
	}
	if target.ProviderID != logicalID {
		return nil, errors.New("provider target does not match this Connexto Proxy adapter")
	}
	if target.ModelID != "" && strings.TrimSpace(target.ModelID) == "" {
		return nil, errors.New("provider model identity must not be empty")
	}

	c.mu.RLock()
	inferenceKey := strings.TrimSpace(c.apiKey)
	inferenceOptional := c.inferenceOptional
	c.mu.RUnlock()
	if logicalID != AntigravityID && !inferenceOptional && inferenceKey == "" {
		return nil, ErrNoCredential
	}

	var requestWritten atomic.Bool
	trace := &httptrace.ClientTrace{WroteRequest: func(info httptrace.WroteRequestInfo) {
		if info.Err == nil {
			requestWritten.Store(true)
		}
	}}
	response, err := c.do(httptrace.WithClientTrace(ctx, trace), http.MethodPost,
		connextoProxyResponses, bytes.NewReader(payload), contentType)
	if err == nil {
		sanitizeConnextoProxyErrorResponse(response)
		return response, nil
	}
	if ctx.Err() != nil || !requestWritten.Load() {
		return response, err
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return nil, ResponseHeaderTimeoutError{}
	}
	return nil, err
}

func sanitizeConnextoProxyErrorResponse(response *http.Response) {
	if response == nil || response.StatusCode < http.StatusMultipleChoices {
		return
	}
	body := connextoProxyErrorBodyForStatus(response.StatusCode)
	if response.Body != nil {
		_ = response.Body.Close()
	}
	response.Body = io.NopCloser(strings.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Type", "application/json")
}

func connextoProxyErrorBodyForStatus(statusCode int) string {
	if statusCode == http.StatusRequestEntityTooLarge {
		return `{"error":{"type":"upstream_error","message":"Connexto Proxy rejected the request body (payload too large). Rebuild the proxy with GATEWAY_JSON_BODY_LIMIT_BYTES (default 32 MiB) or increase the limit."}}`
	}
	return connextoProxyErrorBody
}

func (c *ConnextoProxyClient) CloseIdle() {
	c.httpClient.CloseIdleConnections()
}

// NewConnextoProxyHTTPClient returns the shared transport used by Connexto Proxy logical adapters.
func NewConnextoProxyHTTPClient() *http.Client {
	return &http.Client{Transport: newTransport()}
}

func (c *ConnextoProxyClient) do(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	c.mu.RLock()
	baseURL, apiKey := c.baseURL, c.apiKey
	gwKey := c.resolveGatewayKey()
	c.mu.RUnlock()

	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build provider request: %w", err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if gwKey != "" {
		request.Header.Set("X-Gateway-API-Key", gwKey)
	}
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
		request.Header.Set("x-api-key", apiKey)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("provider request: %w", err)
	}
	return response, nil
}
