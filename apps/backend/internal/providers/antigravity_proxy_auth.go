package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrProxyAntigravityAuthFailed reports a failed Antigravity OAuth proxy call.
var ErrProxyAntigravityAuthFailed = errors.New("connexto proxy antigravity auth request failed")

const proxyAntigravityAuthStatusTimeout = 30 * time.Second
const proxyAntigravityAuthStartTimeout = 120 * time.Second

// AntigravityProxyAuthActiveOperation mirrors optional activeOperation from proxy auth/status.
type AntigravityProxyAuthActiveOperation struct {
	OperationID string  `json:"operationId"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"createdAt,omitempty"`
	ExpiresAt   string  `json:"expiresAt,omitempty"`
	Account     *string `json:"account,omitempty"`
	Error       *string `json:"error,omitempty"`
}

// AntigravityProxyAuthStatusPayload is JSON from GET .../providers/antigravity/auth/status.
type AntigravityProxyAuthStatusPayload struct {
	Authenticated   bool                                 `json:"authenticated"`
	Account         *string                              `json:"account,omitempty"`
	LastCheckedAt   string                               `json:"lastCheckedAt"`
	ViabilityGate   string                               `json:"viabilityGate"`
	Limitations     []string                             `json:"limitations"`
	ActiveOperation *AntigravityProxyAuthActiveOperation `json:"activeOperation,omitempty"`
}

// AntigravityProxyAuthStartPayload is JSON from POST .../providers/antigravity/auth/start.
type AntigravityProxyAuthStartPayload struct {
	OperationID string `json:"operationId"`
	Status      string `json:"status"`
	AuthURL     string `json:"authUrl"`
	State       string `json:"state"`
	ExpiresAt   string `json:"expiresAt"`
}

// AntigravityProxyAuthStartBody is optional JSON for POST .../auth/start.
type AntigravityProxyAuthStartBody struct {
	RedirectURI *string `json:"redirectUri,omitempty"`
}

// GetAntigravityProxyAuthStatus calls GET {endpoint}/providers/antigravity/auth/status on the configured proxy.
// The stored endpoint must include the proxy API prefix (e.g. .../v1), consistent with other Hub ↔ proxy calls.
func (s *Service) GetAntigravityProxyAuthStatus(ctx context.Context, id string) (*AntigravityProxyAuthStatusPayload, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if record.Type != TypeAntigravity {
		return nil, fmt.Errorf("%w: Antigravity auth applies only to antigravity provider", ErrInvalid)
	}
	gatewayKey, err := s.managementCredentialOf(ctx, record)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(gatewayKey) == "" {
		return nil, ErrNoGatewayKey
	}
	base := strings.TrimRight(strings.TrimSpace(record.Endpoint), "/")
	statusURL := base + "/providers/antigravity/auth/status?accountId=" + url.QueryEscape(record.ID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyAntigravityAuthFailed, err)
	}
	req.Header.Set("X-Gateway-API-Key", gatewayKey)
	req.Header.Set("X-Account-ID", record.ID)

	client := &http.Client{Timeout: proxyAntigravityAuthStatusTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyAntigravityAuthFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrProxyAntigravityAuthFailed, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newProxyAntigravityAuthFailed(resp.StatusCode, body)
	}

	var out AntigravityProxyAuthStatusPayload
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("%w: decode: %w", ErrProxyAntigravityAuthFailed, err)
	}
	return &out, nil
}

// StartAntigravityProxyAuth calls POST {endpoint}/providers/antigravity/auth/start on the configured proxy.
// The stored endpoint must include the proxy API prefix (e.g. .../v1), consistent with other Hub ↔ proxy calls.
func (s *Service) StartAntigravityProxyAuth(ctx context.Context, id string, body *AntigravityProxyAuthStartBody) (*AntigravityProxyAuthStartPayload, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if record.Type != TypeAntigravity {
		return nil, fmt.Errorf("%w: Antigravity auth applies only to antigravity provider", ErrInvalid)
	}
	gatewayKey, err := s.managementCredentialOf(ctx, record)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(gatewayKey) == "" {
		return nil, ErrNoGatewayKey
	}
	base := strings.TrimRight(strings.TrimSpace(record.Endpoint), "/")
	startURL := base + "/providers/antigravity/auth/start"

	payloadMap := map[string]string{
		"accountId": record.ID,
	}
	if body != nil && body.RedirectURI != nil && strings.TrimSpace(*body.RedirectURI) != "" {
		payloadMap["redirectUri"] = strings.TrimSpace(*body.RedirectURI)
	}
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyAntigravityAuthFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, startURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyAntigravityAuthFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-API-Key", gatewayKey)
	req.Header.Set("X-Account-ID", record.ID)

	client := &http.Client{Timeout: proxyAntigravityAuthStartTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyAntigravityAuthFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrProxyAntigravityAuthFailed, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newProxyAntigravityAuthFailed(resp.StatusCode, respBody)
	}
	var out AntigravityProxyAuthStartPayload
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("%w: decode: %w", ErrProxyAntigravityAuthFailed, err)
	}
	if strings.TrimSpace(out.AuthURL) == "" {
		return nil, fmt.Errorf("%w: empty authUrl in response", ErrProxyAntigravityAuthFailed)
	}
	return &out, nil
}
