package providers

import (
	"bytes"
	"context"
	"encoding/json"
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
	mu         sync.RWMutex
	baseURL    string
	apiKey     string
	gatewayKey string
	enabled    bool
	httpClient *http.Client
}

// NewConnextoProxyClient builds an adapter with a bounded shared provider transport.
func NewConnextoProxyClient(settings Settings, httpClient *http.Client) (*ConnextoProxyClient, error) {
	if err := config.ValidateBaseURL(settings.BaseURL); err != nil {
		return nil, fmt.Errorf("provider %s: %w", ConnextoProxyID, err)
	}
	if httpClient == nil {
		httpClient = &http.Client{Transport: newTransport()}
	}
	return &ConnextoProxyClient{
		baseURL:    strings.TrimRight(settings.BaseURL, "/"),
		apiKey:     settings.APIKey,
		gatewayKey: settings.ManagementKey,
		enabled:    settings.Enabled,
		httpClient: httpClient,
	}, nil
}

// Reconfigure atomically changes endpoint, credential, and enabled state.
func (c *ConnextoProxyClient) Reconfigure(settings Settings) error {
	if err := config.ValidateBaseURL(settings.BaseURL); err != nil {
		return fmt.Errorf("provider %s: %w", ConnextoProxyID, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = strings.TrimRight(settings.BaseURL, "/")
	c.apiKey = settings.APIKey
	c.gatewayKey = settings.ManagementKey
	c.enabled = settings.Enabled
	return nil
}

// Enabled is false when disabled or when no protected credential was loaded.
func (c *ConnextoProxyClient) Enabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled && c.apiKey != ""
}

// ListModels retrieves Connexto Proxy's model list.
func (c *ConnextoProxyClient) ListModels(ctx context.Context) (*http.Response, error) {
	response, err := c.do(ctx, http.MethodGet, connextoProxyModels, nil, "")
	if err != nil {
		return nil, err
	}
	sanitizeConnextoProxyErrorResponse(response)
	return response, nil
}

// OpenTargetResponses forwards a stateless Responses request without rewriting it.
func (c *ConnextoProxyClient) OpenTargetResponses(ctx context.Context, target ResponsesTarget, payload []byte, contentType, _ string) (*http.Response, error) {
	if target.ProviderID != ConnextoProxyID {
		return nil, errors.New("provider target does not match Connexto Proxy")
	}
	if target.ModelID != "" && strings.TrimSpace(target.ModelID) == "" {
		return nil, errors.New("provider model identity must not be empty")
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
	if response.Body != nil {
		_ = response.Body.Close()
	}
	response.Body = io.NopCloser(strings.NewReader(connextoProxyErrorBody))
	response.ContentLength = int64(len(connextoProxyErrorBody))
	response.Header.Set("Content-Type", "application/json")
}

// FetchModelDescriptors parses claims from Connexto Proxy's catalog.
func (c *ConnextoProxyClient) FetchModelDescriptors(ctx context.Context) ([]ModelDescriptor, error) {
	response, err := c.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return nil, &UpstreamHTTPError{StatusCode: response.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelsResponseBytes+1))
	if err != nil {
		return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
	}
	if len(body) > maxModelsResponseBytes {
		return nil, &DecodeError{StatusCode: response.StatusCode, Err: errors.New("catalog payload exceeds 8 MiB")}
	}

	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Data == nil {
		if err == nil {
			err = errors.New("catalog response has no data array")
		}
		return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
	}

	descriptors := make([]ModelDescriptor, 0, len(envelope.Data))
	seen := make(map[string]struct{}, len(envelope.Data))
	for _, raw := range envelope.Data {
		descriptor, err := connextoProxyDescriptor(raw)
		if err != nil {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
		}
		if _, duplicate := seen[descriptor.ID]; duplicate {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: errors.New("catalog contains duplicate model IDs")}
		}
		seen[descriptor.ID] = struct{}{}
		descriptors = append(descriptors, descriptor)
	}
	return descriptors, nil
}

func connextoProxyDescriptor(raw json.RawMessage) (ModelDescriptor, error) {
	var model struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Object      string `json:"object"`
		OwnedBy     string `json:"owned_by"`
	}
	if err := json.Unmarshal(raw, &model); err != nil {
		return ModelDescriptor{}, err
	}
	if model.ID == "" {
		return ModelDescriptor{}, errors.New("catalog model has no ID")
	}
	name := model.Name
	if name == "" {
		name = model.ID
	}
	descriptor := ModelDescriptor{
		ID:                      model.ID,
		Name:                    name,
		Type:                    model.Object,
		OutputModalities:        []string{"text"},
		OutputModalitiesPresent: true,
		Raw:                     bytes.Clone(raw),
	}
	return descriptor, nil
}

func (c *ConnextoProxyClient) CloseIdle() {
	c.httpClient.CloseIdleConnections()
}

// ProbeConnextoProxyEndpoint verifies service availability without consuming model generation.
// It checks health and/or model listing at the configured endpoint.
func ProbeConnextoProxyEndpoint(ctx context.Context, endpoint, credential string) error {
	client, err := NewConnextoProxyClient(Settings{BaseURL: endpoint, APIKey: credential, Enabled: true}, nil)
	if err != nil {
		return err
	}
	defer client.CloseIdle()
	probeContext, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	response, err := client.ListModels(probeContext)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("connectivity check returned status %d", response.StatusCode)
	}
	return nil
}

func (c *ConnextoProxyClient) resolveGatewayKey() string {
	if c.gatewayKey != "" {
		return c.gatewayKey
	}
	if envKey := os.Getenv("AIHUB_CONNEXTO_PROXY_GATEWAY_KEY"); envKey != "" {
		return envKey
	}
	return "docker-local-gateway-dev"
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

