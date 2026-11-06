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
	c.mu.RLock()
	logicalID := c.logicalProviderID
	c.mu.RUnlock()
	if logicalID == "" {
		logicalID = CursorID
	}

	for _, raw := range envelope.Data {
		descriptor, err := connextoProxyDescriptor(raw)
		if err != nil {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: err}
		}
		switch logicalID {
		case AntigravityID:
			if !connextoProxyModelBelongsToAntigravity(descriptor.ID, raw) {
				continue
			}
		default:
			if connextoProxyModelBelongsToAntigravity(descriptor.ID, raw) {
				continue
			}
		}
		if _, duplicate := seen[descriptor.ID]; duplicate {
			return nil, &DecodeError{StatusCode: response.StatusCode, Err: errors.New("catalog contains duplicate model IDs")}
		}
		seen[descriptor.ID] = struct{}{}
		descriptors = append(descriptors, descriptor)
	}
	return descriptors, nil
}

func rawJSONOwnedBy(raw json.RawMessage) string {
	var probe struct {
		OwnedBy string `json:"owned_by"`
	}
	_ = json.Unmarshal(raw, &probe)
	return probe.OwnedBy
}

func connextoProxyModelBelongsToAntigravity(modelID string, raw json.RawMessage) bool {
	return IsAntigravityCatalogModel(rawJSONOwnedBy(raw), modelID)
}

func connextoProxyDescriptor(raw json.RawMessage) (ModelDescriptor, error) {
	var model struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Object       string `json:"object"`
		OwnedBy      string `json:"owned_by"`
		Modalities   *struct {
			Input *[]string `json:"input"`
		} `json:"modalities"`
		Capabilities struct {
			Tools     *rawSupported `json:"tools"`
			Reasoning *rawSupported `json:"reasoning"`
		} `json:"capabilities"`
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

	antigravity := IsAntigravityCatalogModel(model.OwnedBy, model.ID)

	var toolsSupported, reasoningSupported bool
	if antigravity {
		toolsSupported = model.Capabilities.Tools != nil && model.Capabilities.Tools.Supported != nil && *model.Capabilities.Tools.Supported
		reasoningSupported = model.Capabilities.Reasoning != nil && model.Capabilities.Reasoning.Supported != nil && *model.Capabilities.Reasoning.Supported
	} else {
		// OpenAI-minimal proxy catalogs omit capabilities; Cursor agent models support tools and reasoning.
		toolsSupported = true
		reasoningSupported = true
		if model.Capabilities.Tools != nil && model.Capabilities.Tools.Supported != nil {
			toolsSupported = *model.Capabilities.Tools.Supported
		}
		if model.Capabilities.Reasoning != nil && model.Capabilities.Reasoning.Supported != nil {
			reasoningSupported = *model.Capabilities.Reasoning.Supported
		}
	}

	inputModalities := connextoProxyDefaultInputModalities()
	inputModalitiesPresent := true
	if antigravity {
		inputModalities = []string{"text"}
		inputModalitiesPresent = true
		if model.Modalities != nil && model.Modalities.Input != nil {
			inputModalities = NormalizeInputModalitiesForCatalog(*model.Modalities.Input)
		}
	} else {
		// SPEC E03 §5.4 A2: cursor sync uses vision defaults when upstream omits modalities.input.
		if model.Modalities != nil && model.Modalities.Input != nil {
			inputModalities = NormalizeInputModalitiesForCatalog(*model.Modalities.Input)
			inputModalitiesPresent = true
		}
	}

	descriptor := ModelDescriptor{
		ID:                      model.ID,
		Name:                    name,
		Type:                    model.Object,
		InputModalities:         inputModalities,
		InputModalitiesPresent:  inputModalitiesPresent,
		OutputModalities:        []string{"text"},
		OutputModalitiesPresent: true,
		ToolsSupported:          boolPtr(toolsSupported),
		ReasoningSupported:      boolPtr(reasoningSupported),
		Raw:                     bytes.Clone(raw),
	}
	return descriptor, nil
}

func connextoProxyDefaultInputModalities() []string {
	return DefaultVisionAgentInputModalities()
}

func (c *ConnextoProxyClient) filterModelsListResponse(response *http.Response) (*http.Response, error) {
	if response == nil || response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return response, nil
	}
	c.mu.RLock()
	logicalID := c.logicalProviderID
	c.mu.RUnlock()
	if logicalID == "" {
		logicalID = CursorID
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelsResponseBytes+1))
	if response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("read model list: %w", err)
	}
	if len(body) > maxModelsResponseBytes {
		return nil, errors.New("catalog payload exceeds 8 MiB")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		response.Body = io.NopCloser(strings.NewReader(string(body)))
		response.ContentLength = int64(len(body))
		return response, nil
	}
	rawData, ok := envelope["data"]
	if !ok {
		response.Body = io.NopCloser(strings.NewReader(string(body)))
		response.ContentLength = int64(len(body))
		return response, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(rawData, &items); err != nil {
		response.Body = io.NopCloser(strings.NewReader(string(body)))
		response.ContentLength = int64(len(body))
		return response, nil
	}
	filtered := make([]json.RawMessage, 0, len(items))
	for _, raw := range items {
		var idProbe struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		}
		_ = json.Unmarshal(raw, &idProbe)
		isAnt := IsAntigravityCatalogModel(idProbe.OwnedBy, idProbe.ID)
		switch logicalID {
		case AntigravityID:
			if isAnt {
				filtered = append(filtered, raw)
			}
		default:
			if !isAnt {
				filtered = append(filtered, raw)
			}
		}
	}
	dataBytes, err := json.Marshal(filtered)
	if err != nil {
		return nil, err
	}
	envelope["data"] = dataBytes
	out, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(out))
	response.ContentLength = int64(len(out))
	response.Header.Set("Content-Type", "application/json")
	return response, nil
}

func boolPtr(value bool) *bool {
	return &value
}

func (c *ConnextoProxyClient) CloseIdle() {
	c.httpClient.CloseIdleConnections()
}

// NewConnextoProxyHTTPClient returns the shared transport used by Connexto Proxy logical adapters.
func NewConnextoProxyHTTPClient() *http.Client {
	return &http.Client{Transport: newTransport()}
}

// ProbeCursorEndpoint verifies Cursor provider availability via the internal transport adapter.
func ProbeCursorEndpoint(ctx context.Context, endpoint, credential string) error {
	return ProbeConnextoProxyEndpoint(ctx, endpoint, credential)
}

// ProbeAntigravityEndpoint verifies Antigravity reachability using gateway authentication only.
func ProbeAntigravityEndpoint(ctx context.Context, endpoint, gatewayKey string) error {
	resolved, ok := EffectiveGatewayKeyForAntigravityValidation(gatewayKey)
	if !ok {
		return ErrNoGatewayKey
	}
	client, err := NewConnextoProxyClient(Settings{
		BaseURL:               endpoint,
		ManagementKey:         resolved,
		Enabled:               true,
		LogicalProviderID:     AntigravityID,
		InferenceOptional:     true,
	}, nil)
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

// ProbeConnextoProxyEndpoint verifies service availability without consuming model generation.
// It checks health and/or model listing at the configured endpoint.
func ProbeConnextoProxyEndpoint(ctx context.Context, endpoint, credential string) error {
	client, err := NewConnextoProxyClient(Settings{BaseURL: endpoint, APIKey: credential, Enabled: true, LogicalProviderID: CursorID}, nil)
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
	return resolveConnextoProxyGatewayKeyForTransport(c.gatewayKey)
}

// resolveConnextoProxyGatewayKeyForTransport mirrors proxy local defaults: stored management
// key, then AIHUB_CONNEXTO_PROXY_GATEWAY_KEY, then the docker dev default.
func resolveConnextoProxyGatewayKeyForTransport(stored string) string {
	if strings.TrimSpace(stored) != "" {
		return strings.TrimSpace(stored)
	}
	if envKey := os.Getenv("AIHUB_CONNEXTO_PROXY_GATEWAY_KEY"); strings.TrimSpace(envKey) != "" {
		return strings.TrimSpace(envKey)
	}
	return "docker-local-gateway-dev"
}

// EffectiveGatewayKeyForAntigravityValidation returns a non-empty gateway key for probes when
// either the stored management credential or AIHUB_CONNEXTO_PROXY_GATEWAY_KEY is set.
func EffectiveGatewayKeyForAntigravityValidation(storedManagement string) (string, bool) {
	if strings.TrimSpace(storedManagement) != "" {
		return strings.TrimSpace(storedManagement), true
	}
	if envKey := os.Getenv("AIHUB_CONNEXTO_PROXY_GATEWAY_KEY"); strings.TrimSpace(envKey) != "" {
		return strings.TrimSpace(envKey), true
	}
	return "", false
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

