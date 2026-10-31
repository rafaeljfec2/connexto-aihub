package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"aihub/internal/access"
)

// Gateway groups the access-control settings enforced by internal/access.
type Gateway struct {
	APIKey          string
	AllowedHosts    []string
	AllowedHostsSet bool
	AllowedOrigins  []string
	MaxBodyBytes    int64
}

func defaultGateway() Gateway {
	return Gateway{MaxBodyBytes: access.DefaultMaxBodyBytes}
}

func applyGatewayEnvironment(gateway *Gateway) error {
	if value, ok := os.LookupEnv(EnvGatewayAPIKey); ok {
		gateway.APIKey = value
	}

	if value, ok := os.LookupEnv(EnvGatewayAllowedHosts); ok {
		hosts, err := access.ParseAllowedHosts(value)
		if err != nil {
			return fmt.Errorf("gateway.allowed_hosts (%s): %w", EnvGatewayAllowedHosts, err)
		}

		gateway.AllowedHosts = hosts
		gateway.AllowedHostsSet = true
	}

	if value, ok := os.LookupEnv(EnvGatewayAllowedOrigins); ok {
		origins, err := access.ParseAllowedOrigins(value)
		if err != nil {
			return fmt.Errorf("gateway.allowed_origins (%s): %w", EnvGatewayAllowedOrigins, err)
		}

		gateway.AllowedOrigins = origins
	}

	if value, ok := os.LookupEnv(EnvGatewayMaxBodyBytes); ok {
		limit, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("gateway.max_body_bytes (%s) must be an integer: %w", EnvGatewayMaxBodyBytes, err)
		}

		gateway.MaxBodyBytes = limit
	}

	return nil
}

func (g Gateway) validate(serverHost, providerKey string) []error {
	var errs []error

	if err := access.ValidateAPIKey(g.APIKey); err != nil {
		errs = append(errs, fmt.Errorf("gateway.api_key is invalid: %w", err))
	}

	if providerKey != "" && g.APIKey == providerKey {
		errs = append(errs, errors.New("gateway.api_key must differ from the enabled provider API key"))
	}

	if g.MaxBodyBytes < access.MinBodyBytes || g.MaxBodyBytes > access.MaxBodyBytesCeiling {
		errs = append(errs, fmt.Errorf("gateway.max_body_bytes must be between %d and %d, got %d",
			access.MinBodyBytes, access.MaxBodyBytesCeiling, g.MaxBodyBytes))
	}

	if g.AllowedHostsSet {
		if len(g.AllowedHosts) == 0 {
			errs = append(errs, errors.New("gateway.allowed_hosts must not be empty when set"))
		} else if _, err := access.ParseAllowedHosts(strings.Join(g.AllowedHosts, ",")); err != nil {
			errs = append(errs, fmt.Errorf("gateway.allowed_hosts is invalid: %w", err))
		}
	} else if !isLoopbackServerHost(serverHost) {
		errs = append(errs, errors.New("gateway.allowed_hosts must be set when server.host is non-loopback"))
	}

	if _, err := access.ParseAllowedOrigins(strings.Join(g.AllowedOrigins, ",")); err != nil {
		errs = append(errs, fmt.Errorf("gateway.allowed_origins is invalid: %w", err))
	}

	return errs
}

func isLoopbackServerHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
