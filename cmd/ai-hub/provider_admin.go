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
		providers.TypeOpenCodeGo:    providers.ProbeEndpoint,
		providers.TypeOpenRouter:    providers.ProbeOpenRouterEndpoint,
		providers.TypeCursor:        providers.ProbeCursorEndpoint,
		providers.TypeConnextoProxy: providers.ProbeConnextoProxyEndpoint,
		providers.TypeAntigravity:   providers.ProbeAntigravityEndpoint,
	},
		cfg.Provider.AllowedEndpointHosts)

	service.SetDefaultEndpoint(providers.CursorID, cfg.Provider.ConnextoProxy.BaseURL)
	service.SetDefaultEndpoint(providers.AntigravityID, cfg.Provider.ConnextoProxy.BaseURL)
	service.SetDefaultEndpoint(providers.OpenCodeGoID, cfg.Provider.OpenCodeGo.BaseURL)
	service.SetDefaultEndpoint(providers.OpenRouterID, providers.OpenRouterEndpoint)

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
		{
			ID: providers.AntigravityID, Name: "Antigravity", Type: providers.TypeAntigravity,
			Endpoint:             cfg.Provider.ConnextoProxy.BaseURL,
			ManagementCredential: cfg.Provider.ConnextoProxy.GatewayKey,
			Enabled:              true, WarnIgnoredEnvironment: true,
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
		{Method: http.MethodGet, Pattern: "/api/providers/available", Handler: admin.Available()},
		{Method: http.MethodPost, Pattern: "/api/providers", Handler: admin.Create()},
		{Method: http.MethodGet, Pattern: "/api/providers", Handler: admin.List()},
		{Method: http.MethodGet, Pattern: "/api/providers/{id}", Handler: admin.Get()},
		{Method: http.MethodPatch, Pattern: "/api/providers/{id}", Handler: admin.Update()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/validate", Handler: admin.Validate()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/proxy/runtime/sessions/clear", Handler: admin.ClearAntigravityProxyRuntimeSessions()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/proxy/cursor/runtime/sessions/clear", Handler: admin.ClearCursorUsageProxyRuntimeSessions()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/proxy/cursor/auth/start", Handler: admin.CursorProxyAuthStart()},
		{Method: http.MethodGet, Pattern: "/api/providers/{id}/proxy/cursor/usage/session", Handler: admin.CursorProxyUsageSession()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/proxy/cursor/usage/collect", Handler: admin.CursorProxyUsageCollect()},
		{Method: http.MethodGet, Pattern: "/api/providers/{id}/proxy/cursor/auth/{sessionId}", Handler: admin.CursorProxyAuthStatus()},
		{Method: http.MethodGet, Pattern: "/api/providers/{id}/proxy/antigravity/auth/status", Handler: admin.AntigravityProxyAuthStatus()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/proxy/antigravity/auth/start", Handler: admin.AntigravityProxyAuthStart()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/sync-models", Handler: admin.SyncModels()},
		{Method: http.MethodGet, Pattern: "/api/providers/{id}/status", Handler: admin.Status()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/enable", Handler: admin.Enable()},
		{Method: http.MethodPost, Pattern: "/api/providers/{id}/disable", Handler: admin.Disable()},
		{Method: http.MethodGet, Pattern: "/api/ui/providers/quotas", Handler: admin.Quotas()},
		{Method: http.MethodGet, Pattern: "/api/ui/providers/cursor/usage-analytics", Handler: admin.CursorUsageAnalytics()},
	}
}
