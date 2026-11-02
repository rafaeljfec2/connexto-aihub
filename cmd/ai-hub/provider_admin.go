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
// or seeds configuration before the provider Manager initializes runtime state.
type providerConfiguration struct {
	service *providers.Service
	module  *providers.ConfigModule
}

func buildProviderConfiguration(cfg config.Config, pool *pgxpool.Pool, registry *providers.Registry, logger *slog.Logger) (providerConfiguration, error) {
	cipher, err := providers.NewCredentialCipher(cfg.Credentials.Key)
	if err != nil {
		return providerConfiguration{}, fmt.Errorf("build provider credentials: %w", err)
	}

	service := providers.NewServiceWithProbes(pgstore.New(pool), cipher, registry, map[string]providers.ProbeFunc{
		providers.TypeOpenCodeGo: providers.ProbeEndpoint,
		providers.TypeOpenRouter: providers.ProbeOpenRouterEndpoint,
	},
		cfg.Provider.AllowedEndpointHosts)

	module := providers.NewConfigModuleSet(service, []providers.BootstrapInput{
		{
			ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
			Endpoint: cfg.Provider.OpenCodeGo.BaseURL, Credential: cfg.Provider.OpenCodeGo.APIKey,
			Enabled: true, WarnIgnoredEnvironment: true,
		},
		{
			ID: providers.OpenRouterID, Name: "OpenRouter", Type: providers.TypeOpenRouter,
			Endpoint: providers.OpenRouterEndpoint, Enabled: true,
		},
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
