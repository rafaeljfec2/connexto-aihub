package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrProxySessionClearFailed reports a failed POST to the Connexto Proxy
// runtime session maintenance API.
var ErrProxySessionClearFailed = errors.New("connexto proxy session clear failed")

// ClearAntigravityProxyRuntimeSessions calls POST /v1/runtime/sessions/clear on the
// configured Connexto Proxy with namespace "antigravity", using the stored gateway key.
func (s *Service) ClearAntigravityProxyRuntimeSessions(ctx context.Context, id string) error {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if record.Type != TypeAntigravity {
		return fmt.Errorf("%w: session clearing applies only to antigravity", ErrInvalid)
	}

	gatewayKey, err := s.managementCredentialOf(ctx, record)
	if err != nil {
		return err
	}
	if strings.TrimSpace(gatewayKey) == "" {
		return ErrNoGatewayKey
	}

	base := strings.TrimRight(strings.TrimSpace(record.Endpoint), "/")
	clearURL := base + "/runtime/sessions/clear"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, clearURL, bytes.NewReader([]byte(`{"namespace":"antigravity"}`)))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrProxySessionClearFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-API-Key", gatewayKey)

	client := &http.Client{Timeout: probeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrProxySessionClearFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%w: proxy status %d: %s", ErrProxySessionClearFailed, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	return nil
}
