package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"aihub/internal/access"
	"aihub/internal/catalog"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/metering"
	"aihub/internal/operational"
	"aihub/internal/providers"
	"aihub/internal/records"
	"aihub/internal/server"
)

func buildGatewayRouter(cfg config.Config, logger *slog.Logger, registry *providers.Registry, providerManager *providers.Manager,
	executionManager *execution.Manager, catalogModule *catalog.Manager, recordsModule *records.Service,
	readiness *health.Registry, providerAdmin *gateway.ProviderAdmin, resourceAdmin *gateway.ResourceAdmin,
	concurrencyAdmin *gateway.ConcurrencyAdmin, analyticsAdmin *gateway.AnalyticsAdmin,
	resourceAdmission gateway.ResourceAdmitter, concurrencyGate gateway.ConcurrencyGate, usageMetering *metering.Service,
	eventHub *operational.Hub) (http.Handler, error) {
	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey: cfg.Gateway.APIKey, AdminAPIKey: cfg.Gateway.AdminAPIKey,
		AllowedHosts: cfg.Gateway.AllowedHosts, AllowedHostsExplicit: cfg.Gateway.AllowedHostsSet,
		AllowedOrigins: cfg.Gateway.AllowedOrigins, MaxBodyBytes: cfg.Gateway.MaxBodyBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("build gateway access policy: %w", err)
	}
	routes := []server.Route{
		{Method: http.MethodPost, Pattern: "/v1/responses",
			Handler: gateway.NewResponsesHandlerWithRegistry(registry, executionManager, logger, gateway.StreamInactivityTimeout,
				&gateway.ResponsesRouting{
					Catalog:    catalogModule,
					Statuses:   providerManager.Statuses,
					StaleAfter: cfg.Catalog.StaleAfter,
				}, resourceAdmission, concurrencyGate, usageMetering),
			Admission: access.ResponsesAdmission(policy.MaxBodyBytes())},
		{Method: http.MethodGet, Pattern: "/v1/models", Handler: gateway.NewRegistryModelsHandler(registry, logger)},
		{Method: http.MethodGet, Pattern: "/diagnostics/providers", Handler: gateway.NewProvidersHandler(providerManager.Statuses)},
		{Method: http.MethodGet, Pattern: "/v1/catalog/models", Handler: gateway.NewCatalogModelsHandler(catalogModule)},
		{Method: http.MethodGet, Pattern: "/v1/catalog/models/{provider}/{model...}", Handler: gateway.NewCatalogModelHandler(catalogModule)},
		{Method: http.MethodGet, Pattern: "/v1/catalog/providers", Handler: gateway.NewCatalogProvidersHandler(catalogModule)},
		{Method: http.MethodPost, Pattern: "/v1/catalog/sync",
			Handler: gateway.NewCatalogSyncHandler(catalogModule), Admission: access.CatalogSyncAdmission(policy.MaxBodyBytes())},
		{Method: http.MethodGet, Pattern: "/v1/records/executions", Handler: gateway.NewExecutionRecordsHandler(recordsModule)},
		{Method: http.MethodGet, Pattern: "/v1/records/executions/{execution_id}", Handler: gateway.NewExecutionRecordHandler(recordsModule)},
	}
	router := server.NewRouter(logger, policy, health.Live(), readiness.Ready(),
		append(routes, append(operationalEventsRoutes(eventHub, readiness, providerManager.Statuses, logger),
			append(providerAdminRoutes(providerAdmin),
				append(resourceAdminRoutes(resourceAdmin),
					append(concurrencyAdminRoutes(concurrencyAdmin),
						analyticsAdminRoutes(analyticsAdmin)...)...)...)...)...)...)
	return router, nil
}
