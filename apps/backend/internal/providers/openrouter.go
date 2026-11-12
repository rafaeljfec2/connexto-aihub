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
	"strings"
	"sync"
	"sync/atomic"

	"aihub/internal/config"
)

const (
	OpenRouterID        = "openrouter"
	TypeOpenRouter      = "openrouter"
	OpenRouterEndpoint  = "https://openrouter.ai/api/v1"
	openRouterModels    = "/models"
	openRouterResponses = "/responses"
	openRouterErrorBody = `{"error":{"type":"upstream_error","message":"OpenRouter request failed"}}`
)

// OpenRouterClient implements the OpenAI Responses API through OpenRouter.
// Requests remain opaque so provider-specific model IDs and fields are kept.
type OpenRouterClient struct {
	mu                sync.RWMutex
	baseURL           string
	apiKey            string
	enabled           bool
	logicalProviderID string
	httpClient        *http.Client
}

// NewOpenRouterClient builds an adapter with a bounded shared provider transport.
func NewOpenRouterClient(settings Settings, httpClient *http.Client) (*OpenRouterClient, error) {
	if err := config.ValidateBaseURL(settings.BaseURL); err != nil {
		return nil, fmt.Errorf("provider %s: %w", OpenRouterID, err)
	}
	if httpClient == nil {
		httpClient = &http.Client{Transport: newTransport()}
	}
	return &OpenRouterClient{
		baseURL:           strings.TrimRight(settings.BaseURL, "/"),
		apiKey:            settings.APIKey,
		enabled:           settings.Enabled,
		logicalProviderID: settings.LogicalProviderID,
		httpClient:        httpClient,
	}, nil
}

// Reconfigure atomically changes endpoint, credential, and enabled state.
func (c *OpenRouterClient) Reconfigure(settings Settings) error {
	if err := config.ValidateBaseURL(settings.BaseURL); err != nil {
		return fmt.Errorf("provider %s: %w", OpenRouterID, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = strings.TrimRight(settings.BaseURL, "/")
	c.apiKey = settings.APIKey
	c.enabled = settings.Enabled
	if settings.LogicalProviderID != "" {
		c.logicalProviderID = settings.LogicalProviderID
	}
	return nil
}

func (c *OpenRouterClient) expectedProviderID() string {
	c.mu.RLock()
	id := c.logicalProviderID
	c.mu.RUnlock()
	if id != "" {
		return id
	}
	return OpenRouterID
}

// Enabled is false when disabled or when no protected credential was loaded.
func (c *OpenRouterClient) Enabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled && c.apiKey != ""
}

// ListModels retrieves OpenRouter's provider-scoped model list.
func (c *OpenRouterClient) ListModels(ctx context.Context) (*http.Response, error) {
	response, err := c.do(ctx, http.MethodGet, openRouterModels, nil, "")
	if err != nil {
		return nil, err
	}
	sanitizeOpenRouterErrorResponse(response)
	return response, nil
}

// OpenTargetResponses forwards a stateless Responses request without rewriting it.
func (c *OpenRouterClient) OpenTargetResponses(ctx context.Context, target ResponsesTarget, payload []byte, contentType, _ string) (*http.Response, error) {
	if target.ProviderID != c.expectedProviderID() {
		return nil, errors.New("provider target does not match OpenRouter")
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
		openRouterResponses, bytes.NewReader(payload), contentType)
	if err == nil {
		sanitizeOpenRouterErrorResponse(response)
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

func sanitizeOpenRouterErrorResponse(response *http.Response) {
	if response == nil || response.StatusCode < http.StatusMultipleChoices {
		return
	}
	if response.Body != nil {
		_ = response.Body.Close()
	}
	response.Body = io.NopCloser(strings.NewReader(openRouterErrorBody))
	response.ContentLength = int64(len(openRouterErrorBody))
	response.Header.Set("Content-Type", "application/json")
}

// FetchModelDescriptors parses only claims explicitly present in OpenRouter's catalog.
func (c *OpenRouterClient) FetchModelDescriptors(ctx context.Context) ([]ModelDescriptor, error) {
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
		descriptor, err := openRouterDescriptor(raw)
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

func openRouterDescriptor(raw json.RawMessage) (ModelDescriptor, error) {
	var model struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ContextLength *int64 `json:"context_length"`
		Architecture  struct {
			InputModalities  *[]string `json:"input_modalities"`
			OutputModalities *[]string `json:"output_modalities"`
		} `json:"architecture"`
		TopProvider struct {
			ContextLength       *int64 `json:"context_length"`
			MaxCompletionTokens *int64 `json:"max_completion_tokens"`
		} `json:"top_provider"`
		SupportedParameters []string `json:"supported_parameters"`
	}
	if err := json.Unmarshal(raw, &model); err != nil {
		return ModelDescriptor{}, err
	}
	if model.ID == "" {
		return ModelDescriptor{}, errors.New("catalog model has no ID")
	}

	limitContext := model.ContextLength
	if limitContext == nil && model.TopProvider.ContextLength != nil {
		limitContext = model.TopProvider.ContextLength
	}

	descriptor := ModelDescriptor{
		ID:           model.ID,
		Name:         model.Name,
		LimitContext: limitContext,
		LimitOutput:  model.TopProvider.MaxCompletionTokens,
		Raw:          bytes.Clone(raw),
	}
	if model.Architecture.InputModalities != nil {
		descriptor.InputModalities = NormalizeInputModalitiesForCatalog(*model.Architecture.InputModalities)
		descriptor.InputModalitiesPresent = true
	}
	if model.Architecture.OutputModalities != nil {
		descriptor.OutputModalities = *model.Architecture.OutputModalities
		descriptor.OutputModalitiesPresent = true
	}

	if model.SupportedParameters != nil {
		var tools, reasoning, structured bool
		for _, param := range model.SupportedParameters {
			switch param {
			case "tools", "tool_choice":
				tools = true
			case "reasoning", "include_reasoning", "reasoning_effort":
				reasoning = true
			case "structured_outputs", "response_format":
				structured = true
			}
		}
		descriptor.ToolsSupported = boolPtr(tools)
		descriptor.ReasoningSupported = boolPtr(reasoning)
		descriptor.StructuredOutput = boolPtr(structured)
	}

	return descriptor, nil
}

func (c *OpenRouterClient) CloseIdle() {
	c.httpClient.CloseIdleConnections()
}

// ProbeOpenRouterEndpoint verifies credentials through the model-list endpoint.
func ProbeOpenRouterEndpoint(ctx context.Context, endpoint, credential string) error {
	client, err := NewOpenRouterClient(Settings{BaseURL: endpoint, APIKey: credential, Enabled: true}, nil)
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

func (c *OpenRouterClient) do(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	c.mu.RLock()
	baseURL, apiKey := c.baseURL, c.apiKey
	c.mu.RUnlock()

	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build provider request: %w", err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("provider request: %w", err)
	}
	return response, nil
}
