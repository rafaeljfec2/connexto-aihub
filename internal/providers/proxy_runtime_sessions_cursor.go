package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CursorUsageProxyRuntimeSessionClearResult is JSON from Connexto Proxy POST /v1/runtime/sessions/clear.
type CursorUsageProxyRuntimeSessionClearResult struct {
	Cleared   bool   `json:"cleared"`
	Namespace string `json:"namespace"`
}

// ClearCursorUsageProxyRuntimeSessions calls POST /v1/runtime/sessions/clear with namespace "cursor-usage".
func (s *Service) ClearCursorUsageProxyRuntimeSessions(ctx context.Context, id string) (*CursorUsageProxyRuntimeSessionClearResult, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if record.Type != TypeCursor {
		return nil, fmt.Errorf("%w: cursor-usage clearing applies only to cursor", ErrInvalid)
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
	clearURL := base + "/runtime/sessions/clear"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, clearURL, bytes.NewReader([]byte(`{"namespace":"cursor-usage"}`)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxySessionClearFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-API-Key", resolved)
	client := &http.Client{Timeout: proxyRuntimeSessionsClearTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxySessionClearFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("%w: proxy status %d: %s", ErrProxySessionClearFailed, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrProxySessionClearFailed, err)
	}
	var out CursorUsageProxyRuntimeSessionClearResult
	if err := json.Unmarshal(body, &out); err != nil {
		return &CursorUsageProxyRuntimeSessionClearResult{Cleared: true, Namespace: "cursor-usage"}, nil
	}
	if out.Namespace == "" {
		out.Namespace = "cursor-usage"
	}
	return &out, nil
}
