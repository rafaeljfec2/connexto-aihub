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

// ErrProxySessionClearFailed reports a failed POST to the Connexto Proxy
// runtime session maintenance API.
var ErrProxySessionClearFailed = errors.New("connexto proxy session clear failed")

// proxyRuntimeSessionsClearTimeout allows Antigravity session/list + session/close.
const proxyRuntimeSessionsClearTimeout = 120 * time.Second

// AntigravityProxyRuntimeSessionClearResult is the JSON body from Connexto Proxy
// POST /v1/runtime/sessions/clear when namespace is antigravity (extra fields optional).
type AntigravityProxyRuntimeSessionClearResult struct {
	Cleared                  bool     `json:"cleared"`
	Namespace                string   `json:"namespace"`
	MappedAcpIDsCount        *int     `json:"mapped_acp_ids_count,omitempty"`
	ListedAcpSessionsCount   *int     `json:"listed_acp_sessions_count,omitempty"`
	AcpSessionsClosed        *int     `json:"acp_sessions_closed,omitempty"`
	AcpCloseFailures         *int     `json:"acp_close_failures,omitempty"`
	Warnings                 []string `json:"warnings,omitempty"`
}

// ClearAntigravityProxyRuntimeSessions calls POST /v1/runtime/sessions/clear on the
// configured Connexto Proxy with namespace "antigravity", using the stored gateway key.
func (s *Service) ClearAntigravityProxyRuntimeSessions(ctx context.Context, id string) (*AntigravityProxyRuntimeSessionClearResult, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if record.Type != TypeAntigravity {
		return nil, fmt.Errorf("%w: session clearing applies only to antigravity", ErrInvalid)
	}

	gatewayKey, err := s.managementCredentialOf(ctx, record)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(gatewayKey) == "" {
		return nil, ErrNoGatewayKey
	}

	base := strings.TrimRight(strings.TrimSpace(record.Endpoint), "/")
	clearURL := base + "/runtime/sessions/clear"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, clearURL, bytes.NewReader([]byte(`{"namespace":"antigravity"}`)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProxySessionClearFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-API-Key", gatewayKey)

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
	var out AntigravityProxyRuntimeSessionClearResult
	if err := json.Unmarshal(body, &out); err != nil {
		return &AntigravityProxyRuntimeSessionClearResult{Cleared: true, Namespace: "antigravity"}, nil
	}
	if out.Namespace == "" {
		out.Namespace = "antigravity"
	}
	if !out.Cleared {
		out.Cleared = true
	}
	return &out, nil
}
