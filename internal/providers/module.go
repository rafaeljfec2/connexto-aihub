package providers

import (
	"context"
	"errors"
	"io"
	"log/slog"
)

// ConfigModuleName identifies the provider configuration bootstrap module.
const ConfigModuleName = "provider-config"

// ConfigModule loads the persisted provider configuration at startup, seeding
// it from the environment on first boot. It must start after database
// migrations and before the provider Manager initializes runtime state.
type ConfigModule struct {
	service    *Service
	bootstraps []BootstrapInput
	logger     *slog.Logger
}

// NewConfigModule builds the bootstrap module.
func NewConfigModule(service *Service, bootstrap BootstrapInput, logger *slog.Logger) *ConfigModule {
	bootstrap.WarnIgnoredEnvironment = true
	return NewConfigModuleSet(service, []BootstrapInput{bootstrap}, logger)
}

// NewConfigModuleSet builds one startup module for several provider configs.
func NewConfigModuleSet(service *Service, bootstraps []BootstrapInput, logger *slog.Logger) *ConfigModule {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &ConfigModule{service: service, bootstraps: append([]BootstrapInput(nil), bootstraps...), logger: logger}
}

// Name implements app.Module.
func (m *ConfigModule) Name() string {
	return ConfigModuleName
}

// Start seeds or loads the configuration and applies it to the runtime. A
// failure fails startup: running with an unknown provider state is unsafe.
func (m *ConfigModule) Start(ctx context.Context) error {
	for _, bootstrap := range m.bootstraps {
		if err := m.startProvider(ctx, bootstrap); err != nil {
			return err
		}
	}
	return nil
}

func (m *ConfigModule) startProvider(ctx context.Context, bootstrap BootstrapInput) error {
	persisted, persistedErr := m.service.Get(ctx, bootstrap.ID)
	if persistedErr != nil && !errors.Is(persistedErr, ErrNotFound) {
		return persistedErr
	}
	record, err := m.service.Bootstrap(ctx, bootstrap)
	if err != nil {
		return err
	}
	if persistedErr == nil && bootstrap.WarnIgnoredEnvironment {
		m.warnIgnoredEnvironment(bootstrap, persisted)
	}
	m.logger.Info("provider configuration loaded", "provider", record.ID,
		"enabled", record.Enabled, "credential_configured", record.HasCredential)
	return nil
}

// warnIgnoredEnvironment reports environment settings that the persisted
// configuration overrides. It logs whether they were present, never their
// values: the credential is never written to logs.
func (m *ConfigModule) warnIgnoredEnvironment(bootstrap BootstrapInput, persisted Record) {
	endpointDiffers := bootstrap.Endpoint != persisted.Endpoint
	credentialSet := bootstrap.Credential != ""

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
