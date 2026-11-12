// Package providers implements the concrete OpenCode Go provider integration:
// credential handling, model catalog retrieval, Responses execution and the
// provider availability state machine.
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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"aihub/internal/config"
)

// OpenCodeGoID identifies the first integrated provider.
const OpenCodeGoID = "opencode-go"

// ResponsesTarget identifies the already-selected destination for a request.
type ResponsesTarget struct {
	ProviderID string
	ModelID    string
}

// ResponseHeaderTimeoutError marks the configured upstream header deadline.
type ResponseHeaderTimeoutError struct{}

func (ResponseHeaderTimeoutError) Error() string {
	return "provider response header timeout"
}

// IsResponseHeaderTimeout identifies this transport-specific opening timeout.
func (ResponseHeaderTimeoutError) IsResponseHeaderTimeout() bool {
	return true
}

// Endpoint paths appended to the configured base URL. They follow the spike
// findings (docs/research/opencode-go-protocol-spike.md §2.1): the models path
// deliberately differs from the Responses path.
const (
	responsesPath = "/openai/v1/responses"
	modelsPath    = "/v1/models"
)

const (
	dialTimeout           = 10 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 120 * time.Second
)

// Client executes requests against the OpenCode Go upstream. It is concrete on
// purpose: the spike validated a single adapter shape; abstractions enter when
// a second provider does.
type Client struct {
	mu                sync.RWMutex
	baseURL           string
	apiKey            string
	enabled           bool
	logicalProviderID string
	httpClient        *http.Client
}

// Settings is the runtime configuration of the adapter. Changing it never
// cancels executions that were already admitted; it only affects new requests.
type Settings struct {
	BaseURL       string
	APIKey        string
	ManagementKey string
	Enabled       bool
	// ProviderType identifies the adapter protocol type (opencode-go, openrouter, cursor, antigravity).
	ProviderType string
	// LogicalProviderID is the registry id for the configured account (e.g. cursor, cursor-work).
	LogicalProviderID string
	// InferenceOptional is true when upstream execution does not require a separate inference API key
	// (gateway authentication only); the adapter stays enabled while disabled=false.
	InferenceOptional bool
}

// NewClient builds the provider client. A nil httpClient selects the package
// transport with bounded dial, handshake and header timeouts.
func NewClient(cfg config.ProviderOpenCodeGo, httpClient *http.Client) (*Client, error) {
	if err := config.ValidateBaseURL(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("provider %s: %w", OpenCodeGoID, err)
	}

	if httpClient == nil {
		httpClient = &http.Client{Transport: newTransport()}
	}

	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		enabled:    true,
		httpClient: httpClient,
	}, nil
}

// Reconfigure replaces the endpoint, credential and enabled state at runtime.
// The base URL is validated with the same rules as the startup configuration.
func (c *Client) Reconfigure(settings Settings) error {
	if err := config.ValidateBaseURL(settings.BaseURL); err != nil {
		return fmt.Errorf("provider %s: %w", OpenCodeGoID, err)
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

func (c *Client) expectedProviderID() string {
	c.mu.RLock()
	id := c.logicalProviderID
	c.mu.RUnlock()
	if id != "" {
		return id
	}
	return OpenCodeGoID
}

// Enabled reports whether a credential is configured. A disabled client
// refuses no request path but must never perform upstream calls with an empty
// Authorization header.
func (c *Client) Enabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.enabled && c.apiKey != ""
}

// ListModels retrieves the model catalog from the vendor-specific path.
func (c *Client) ListModels(ctx context.Context) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, modelsPath, nil, nil)
}

// OpenResponses forwards a Responses API request body verbatim. The adapter
// only supplies transport headers; it never parses or rewrites the payload.
func (c *Client) OpenResponses(ctx context.Context, body io.Reader, contentType, sessionID string) (*http.Response, error) {
	header := make(http.Header, 2)
	header.Set("Content-Type", contentType)
	header.Set("X-Opencode-Session", sessionID)

	return c.do(ctx, http.MethodPost, responsesPath, body, header)
}

// OpenTargetResponses forwards a prepared payload for an explicit provider
// target. Target metadata never rewrites the payload.
func (c *Client) OpenTargetResponses(ctx context.Context, target ResponsesTarget, payload []byte, contentType, sessionID string) (*http.Response, error) {
	if target.ProviderID != c.expectedProviderID() {
		return nil, errors.New("provider target does not match OpenCode Go")
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
	requestContext := httptrace.WithClientTrace(ctx, trace)
	header := make(http.Header, 2)
	header.Set("Content-Type", contentType)
	header.Set("X-Opencode-Session", sessionID)

	response, err := c.do(requestContext, http.MethodPost, responsesPath,
		bytes.NewReader(payload), header)
	if err == nil || ctx.Err() != nil || !requestWritten.Load() {
		return response, err
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return nil, ResponseHeaderTimeoutError{}
	}
	return nil, err
}

// Stop releases idle connections for process shutdown.
func (c *Client) CloseIdle() {
	c.httpClient.CloseIdleConnections()
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, header http.Header) (*http.Response, error) {
	c.mu.RLock()
	baseURL, apiKey := c.baseURL, c.apiKey
	c.mu.RUnlock()

	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build provider request: %w", err)
	}

	if header != nil {
		request.Header = header
	}

	// The credential never travels in the URL or in an error message.
	request.Header.Set("Authorization", "Bearer "+apiKey)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("provider request: %w", err)
	}

	return response, nil
}

func newTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
