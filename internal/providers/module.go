package providers

import (
	"context"
	"errors"
	"log/slog"
)

// ConfigModuleName identifies the provider configuration bootstrap module.
const ConfigModuleName = "provider-config"

// ConfigModule loads the persisted provider configuration at startup, seeding
// it from the environment on first boot. It must start after the database
// migrations and before the provider module verifies connectivity.
type ConfigModule struct {
	service   *Service
	bootstrap BootstrapInput
	logger    *slog.Logger
}

// NewConfigModule builds the bootstrap module.
func NewConfigModule(service *Service, bootstrap BootstrapInput, logger *slog.Logger) *ConfigModule {
	return &ConfigModule{service: service, bootstrap: bootstrap, logger: logger}
}

// Name implements app.Module.
func (m *ConfigModule) Name() string {
	return ConfigModuleName
}

// Start seeds or loads the configuration and applies it to the runtime. A
// failure fails startup: running with an unknown provider state is unsafe.
func (m *ConfigModule) Start(ctx context.Context) error {
	persisted, persistedErr := m.service.Get(ctx, m.bootstrap.ID)
	if persistedErr != nil && !errors.Is(persistedErr, ErrNotFound) {
		return persistedErr
	}

	record, err := m.service.Bootstrap(ctx, m.bootstrap)
	if err != nil {
		return err
	}

	if persistedErr == nil {
		m.warnIgnoredEnvironment(persisted)
	}

	m.logger.Info("provider configuration loaded", "provider", record.ID,
		"enabled", record.Enabled, "credential_configured", record.HasCredential)

	return nil
}

// warnIgnoredEnvironment reports environment settings that the persisted
// configuration overrides. It logs whether they were present, never their
// values: the credential is never written to logs.
func (m *ConfigModule) warnIgnoredEnvironment(persisted Record) {
	endpointDiffers := m.bootstrap.Endpoint != persisted.Endpoint
	credentialSet := m.bootstrap.Credential != ""

	if !endpointDiffers && !credentialSet {
		return
	}

	m.logger.Warn("environment provider settings ignored; persisted configuration is authoritative",
		"provider", persisted.ID, "endpoint_differs", endpointDiffers,
		"credential_in_environment", credentialSet)
}

// Stop has nothing to release.
func (m *ConfigModule) Stop(_ context.Context) error {
	return nil
}
