package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/providers/pgstore"
	"aihub/internal/server"
)

// providerConfiguration groups the PostgreSQL-backed provider administration
// wiring: the service used by the admin API and the startup module that loads
// or seeds the configuration before the provider verifies connectivity.
type providerConfiguration struct {
	service *providers.Service
	module  *providers.ConfigModule
}

func buildProviderConfiguration(cfg config.Config, pool *pgxpool.Pool, client *providers.Client, logger *slog.Logger) (providerConfiguration, error) {
	cipher, err := providers.NewCredentialCipher(cfg.Credentials.Key)
	if err != nil {
		return providerConfiguration{}, fmt.Errorf("build provider credentials: %w", err)
	}

	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, client)

	service := providers.NewService(pgstore.New(pool), cipher, registry, providers.ProbeEndpoint)

	module := providers.NewConfigModule(service, providers.BootstrapInput{
		ID:         providers.OpenCodeGoID,
		Name:       "OpenCode Go",
		Type:       providers.TypeOpenCodeGo,
		Endpoint:   cfg.Provider.OpenCodeGo.BaseURL,
		Credential: cfg.Provider.OpenCodeGo.APIKey,
		Enabled:    true,
	}, logger)

	return providerConfiguration{service: service, module: module}, nil
}

// providerAdminRoutes returns the administrative routes, or none when the
// provider administration is unavailable (file persistence mode).
func providerAdminRoutes(admin *gateway.ProviderAdmin) []server.Route {
	if admin == nil {
		return nil
	}

	return []server.Route{
		{Method: http.MethodPost, Pattern: "/api/providers", Handler: admin.Create()},
		{Method: http.MethodGet, Pattern: "/api/providers", Handler: admin.List()},
		{Method: http.MethodGet, Pattern: "/api/providers/{id}", Handler: admin.Get()},
		{Method: http.MethodPatch, Pattern: "/api/providers/{id}", Handler: admin.Update()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/validate", Handler: admin.Validate()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/enable", Handler: admin.Enable()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/disable", Handler: admin.Disable()},
	}
}
