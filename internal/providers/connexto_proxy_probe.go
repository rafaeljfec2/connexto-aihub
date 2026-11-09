package providers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
)

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
		BaseURL:           endpoint,
		ManagementKey:     resolved,
		Enabled:           true,
		LogicalProviderID: AntigravityID,
		InferenceOptional: true,
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

