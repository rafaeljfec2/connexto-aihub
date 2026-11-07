package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrProxyCursorUsageFailed = errors.New("connexto proxy cursor usage request failed")

const (
	proxyCursorUsageCollectTimeout = 120 * time.Second
	proxyCursorAuthStartTimeout    = 120 * time.Second
	proxyCursorAuthStatusTimeout   = 30 * time.Second
)

type CursorProxyUsageCollectBody struct {
	AccountID string `json:"accountId"`
}

type CursorProxyUsageCollectResponse struct {
	Status      string                 `json:"status"`
	Provider    string                 `json:"provider"`
	AccountID   string                 `json:"accountId"`
	CollectedAt string                 `json:"collectedAt,omitempty"`
	Usage       map[string]interface{} `json:"usage,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CursorProxyAuthStartBody struct {
	AccountID string `json:"accountId"`
}

type CursorProxyAuthStartResponse struct {
	Status         string `json:"status"`
	SessionID      string `json:"sessionId"`
	ExpiresAt      string `json:"expiresAt"`
	InteractionURL string `json:"interactionUrl"`
}

type CursorProxyAuthStatusResponse struct {
	SessionID string `json:"sessionId"`
	AccountID string `json:"accountId"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expiresAt"`
}

func (s *Service) PostCursorProxyUsageCollect(ctx context.Context, providerID string) (*CursorProxyUsageCollectResponse, error) {
	record, err := s.repo.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if record.Type != TypeCursor {
		return nil, fmt.Errorf("%w: cursor usage collect applies only to cursor provider", ErrInvalid)
	}
	gatewayKey, err := s.managementCredentialOf(ctx, record)
	if err != nil {
		return nil, err
	}
	resolved, ok := EffectiveGatewayKeyForAntigravityValidation(gatewayKey)
	if !ok {
		return nil, ErrNoGatewayKey
	}
	base := strings.TrimRight(strings.TrimSpace(record.Endpoint), "/")
	url := base + "/providers/cursor/usage/collect"

	payload, err := json.Marshal(CursorProxyUsageCollectBody{
		AccountID: providerID,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyCursorUsageFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyCursorUsageFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-API-Key", resolved)
	req.Header.Set("X-Hub-Account-Id", providerID)

	client := &http.Client{Timeout: proxyCursorUsageCollectTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyCursorUsageFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrProxyCursorUsageFailed, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newProxyCursorUsageFailed(resp.StatusCode, body)
	}

	var out CursorProxyUsageCollectResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("%w: decode: %w", ErrProxyCursorUsageFailed, err)
	}
	return &out, nil
}

func (s *Service) StartCursorProxyAuth(ctx context.Context, providerID string) (*CursorProxyAuthStartResponse, error) {
	record, err := s.repo.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if record.Type != TypeCursor {
		return nil, fmt.Errorf("%w: cursor auth applies only to cursor provider", ErrInvalid)
	}
	gatewayKey, err := s.managementCredentialOf(ctx, record)
	if err != nil {
		return nil, err
	}
	resolved, ok := EffectiveGatewayKeyForAntigravityValidation(gatewayKey)
	if !ok {
		return nil, ErrNoGatewayKey
	}
	base := strings.TrimRight(strings.TrimSpace(record.Endpoint), "/")
	url := base + "/providers/cursor/auth"

	payload, err := json.Marshal(CursorProxyAuthStartBody{AccountID: providerID})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyCursorUsageFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyCursorUsageFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-API-Key", resolved)
	req.Header.Set("X-Hub-Account-Id", providerID)

	client := &http.Client{Timeout: proxyCursorAuthStartTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyCursorUsageFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrProxyCursorUsageFailed, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newProxyCursorUsageFailed(resp.StatusCode, body)
	}

	var out CursorProxyAuthStartResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("%w: decode: %w", ErrProxyCursorUsageFailed, err)
	}
	return &out, nil
}

func (s *Service) GetCursorProxyAuthStatus(ctx context.Context, providerID, sessionID string) (*CursorProxyAuthStatusResponse, error) {
	record, err := s.repo.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if record.Type != TypeCursor {
		return nil, fmt.Errorf("%w: cursor auth applies only to cursor provider", ErrInvalid)
	}
	gatewayKey, err := s.managementCredentialOf(ctx, record)
	if err != nil {
		return nil, err
	}
	resolved, ok := EffectiveGatewayKeyForAntigravityValidation(gatewayKey)
	if !ok {
		return nil, ErrNoGatewayKey
	}
	base := strings.TrimRight(strings.TrimSpace(record.Endpoint), "/")
	url := base + "/providers/cursor/auth/" + sessionID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyCursorUsageFailed, err)
	}
	req.Header.Set("X-Gateway-API-Key", resolved)
	req.Header.Set("X-Hub-Account-Id", providerID)

	client := &http.Client{Timeout: proxyCursorAuthStatusTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxyCursorUsageFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrProxyCursorUsageFailed, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newProxyCursorUsageFailed(resp.StatusCode, body)
	}

	var out CursorProxyAuthStatusResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("%w: decode: %w", ErrProxyCursorUsageFailed, err)
	}
	if out.AccountID != "" && out.AccountID != providerID {
		return nil, fmt.Errorf("%w: session account mismatch", ErrProxyCursorUsageFailed)
	}
	return &out, nil
}
