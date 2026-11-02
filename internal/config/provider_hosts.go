package config

import (
	"errors"
	"fmt"
	"os"

	"aihub/internal/access"
)

// EnvProviderAllowedHosts names the exact host allowlist for provider endpoints
// configured through the administrative API. The provider credential is only
// ever sent to these hosts.
const EnvProviderAllowedHosts = "AIHUB_PROVIDER_ALLOWED_HOSTS"

// defaultProviderAllowedHosts is the production OpenCode Go host.
var defaultProviderAllowedHosts = []string{"opencode.ai", "openrouter.ai"}

// applyProviderHostsEnvironment reads the endpoint allowlist. Entries are exact
// DNS names or IP literals, without ports, schemes, paths or wildcards.
func applyProviderHostsEnvironment(cfg *Config) error {
	value, ok := os.LookupEnv(EnvProviderAllowedHosts)
	if !ok {
		return nil
	}

	hosts, err := access.ParseAllowedHosts(value)
	if err != nil {
		return fmt.Errorf("%s: %w", EnvProviderAllowedHosts, err)
	}

	cfg.Provider.AllowedEndpointHosts = hosts

	return nil
}

// validateProviderHosts requires a non-empty allowlist.
func validateProviderHosts(c Config) []error {
	if len(c.Provider.AllowedEndpointHosts) == 0 {
		return []error{errors.New(EnvProviderAllowedHosts + " must list at least one host")}
	}

	return nil
}
