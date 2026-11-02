package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"aihub/internal/access"
)

// EnvCredentialKey names the base64-encoded AES-256 key that encrypts provider
// credentials stored in PostgreSQL. The key lives outside the database.
const EnvCredentialKey = "AIHUB_CREDENTIAL_KEY"

// CredentialKeyBytes is the AES-256 key length.
const CredentialKeyBytes = 32

// Credentials groups the provider credential protection settings.
type Credentials struct {
	Key []byte
}

// applyCredentialEnvironment decodes the credential key. An empty value is
// treated as unset. The decoded key is never echoed in an error.
func applyCredentialEnvironment(cfg *Config) error {
	value, ok := os.LookupEnv(EnvCredentialKey)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}

	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("%s must be base64-encoded: %w", EnvCredentialKey, err)
	}

	if len(key) != CredentialKeyBytes {
		return fmt.Errorf("%s must decode to %d bytes, got %d", EnvCredentialKey, CredentialKeyBytes, len(key))
	}

	cfg.Credentials.Key = key

	return nil
}

// validateAdminAPIKey requires the administrative key in PostgreSQL mode, where
// the administrative API under /api/ exists. It must be a valid API key
// distinct from the gateway and provider keys.
func validateAdminAPIKey(c Config) []error {
	if c.Catalog.Persistence != EnvCatalogPersistencePostgres {
		return nil
	}

	if c.Gateway.AdminAPIKey == "" {
		return []error{errors.New(EnvAdminAPIKey + " is required when catalog persistence is postgres")}
	}

	var errs []error

	if err := access.ValidateAPIKey(c.Gateway.AdminAPIKey); err != nil {
		errs = append(errs, fmt.Errorf("%s is invalid: %w", EnvAdminAPIKey, err))
	}

	if c.Gateway.AdminAPIKey == c.Gateway.APIKey {
		errs = append(errs, errors.New(EnvAdminAPIKey+" must differ from "+EnvGatewayAPIKey))
	}

	if providerKey := c.Provider.OpenCodeGo.APIKey; providerKey != "" && c.Gateway.AdminAPIKey == providerKey {
		errs = append(errs, errors.New(EnvAdminAPIKey+" must differ from the provider API key"))
	}

	return errs
}

// validateCredentials requires the credential key whenever provider state is
// persisted in PostgreSQL: stored credentials must be decryptable after a
// restart, and failing at startup is safer than starting without the key.
func validateCredentials(c Config) []error {
	if c.Catalog.Persistence == EnvCatalogPersistencePostgres && len(c.Credentials.Key) == 0 {
		return []error{errors.New(EnvCredentialKey + " is required when catalog persistence is postgres")}
	}

	return nil
}
