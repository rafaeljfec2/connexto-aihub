package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"aihub/internal/access"
	"aihub/internal/catalog"
	"aihub/internal/config"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/generations"
	"aihub/internal/health"
	"aihub/internal/metering"
	"aihub/internal/operational"
	"aihub/internal/providers"
	"aihub/internal/records"
	"aihub/internal/server"
)

func buildGatewayRouter(cfg config.Config, logger *slog.Logger, registry *providers.Registry, providerManager *providers.Manager,
	providerService *providers.Service,
	executionManager *execution.Manager, catalogModule *catalog.Manager, recordsModule *records.Service,
	readiness *health.Registry, providerAdmin *gateway.ProviderAdmin, resourceAdmin *gateway.ResourceAdmin,
	concurrencyAdmin *gateway.ConcurrencyAdmin, analyticsAdmin *gateway.AnalyticsAdmin,
	resourceAdmission gateway.ResourceAdmitter, concurrencyGate gateway.ConcurrencyGate, usageMetering *metering.Service,
	eventHub *operational.Hub, uiAuth UIAuthBundle, quotaService *providers.QuotaService) (http.Handler, error) {
	return buildGatewayRouterWithGenerations(cfg, logger, registry, providerManager, providerService, executionManager,
		catalogModule, recordsModule, generations.NewService(recordsModule, nil, logger), readiness,
		providerAdmin, resourceAdmin, concurrencyAdmin, analyticsAdmin, resourceAdmission,
		concurrencyGate, usageMetering, eventHub, uiAuth, quotaService)
}

func buildGatewayRouterWithGenerations(cfg config.Config, logger *slog.Logger, registry *providers.Registry, providerManager *providers.Manager,
	providerService *providers.Service,
	executionManager *execution.Manager, catalogModule *catalog.Manager, recordsModule *records.Service,
	generationService *generations.Service, readiness *health.Registry, providerAdmin *gateway.ProviderAdmin,
	resourceAdmin *gateway.ResourceAdmin, concurrencyAdmin *gateway.ConcurrencyAdmin,
	analyticsAdmin *gateway.AnalyticsAdmin, resourceAdmission gateway.ResourceAdmitter,
	concurrencyGate gateway.ConcurrencyGate, usageMetering *metering.Service,
	eventHub *operational.Hub, uiAuth UIAuthBundle, quotaService *providers.QuotaService) (http.Handler, error) {
	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey: cfg.Gateway.APIKey, AdminAPIKey: cfg.Gateway.AdminAPIKey,
		AllowedHosts: cfg.Gateway.AllowedHosts, AllowedHostsExplicit: cfg.Gateway.AllowedHostsSet,
		AllowedOrigins: cfg.Gateway.AllowedOrigins, MaxBodyBytes: cfg.Gateway.MaxBodyBytes,
		Operator: uiAuth.OperatorGate,
	})
	if err != nil {
		return nil, fmt.Errorf("build gateway access policy: %w", err)
	}
	routingDeps := &gateway.ResponsesRouting{
		Catalog:         catalogModule,
		Statuses:        providerManager.Statuses,
		ProviderService: providerService,
		StaleAfter:      cfg.Catalog.StaleAfter,
		QuotaStaleTTL:   2 * time.Minute,
	}
	if quotaService != nil {
		routingDeps.QuotaSnapshots = func(ctx context.Context) []providers.ProviderQuotaSnapshot {
			resp, err := quotaService.GetSnapshots(ctx)
			if err != nil {
				return nil
			}
			return resp.Snapshots
		}
	}
	responsesHandler := gateway.NewResponsesHandlerWithRegistryAndGenerations(registry, executionManager, logger, gateway.StreamInactivityTimeout,
		routingDeps, resourceAdmission, concurrencyGate, usageMetering, generationService)
	chatCompletionsHandler := gateway.NewChatCompletionsHandler(responsesHandler)
	routes := []server.Route{
		{Method: http.MethodPost, Pattern: "/v1/responses",
			Handler: responsesHandler, Admission: access.ResponsesAdmission(policy.MaxBodyBytes())},
		{Method: http.MethodPost, Pattern: "/v1/chat/completions",
			Handler: chatCompletionsHandler, Admission: access.ResponsesAdmission(policy.MaxBodyBytes())},
		{Method: http.MethodGet, Pattern: "/v1/models", Handler: gateway.NewRegistryModelsHandler(registry, logger)},
		{Method: http.MethodGet, Pattern: "/diagnostics/providers", Handler: gateway.NewProvidersHandler(providerManager.Statuses)},
		{Method: http.MethodGet, Pattern: "/v1/catalog/models", Handler: gateway.NewCatalogModelsHandler(catalogModule)},
		{Method: http.MethodGet, Pattern: "/v1/catalog/models/{provider}/{model...}", Handler: gateway.NewCatalogModelHandler(catalogModule)},
		{Method: http.MethodGet, Pattern: "/v1/catalog/providers", Handler: gateway.NewCatalogProvidersHandler(catalogModule)},
		{Method: http.MethodPost, Pattern: "/v1/catalog/sync",
			Handler: gateway.NewCatalogSyncHandler(catalogModule), Admission: access.CatalogSyncAdmission(policy.MaxBodyBytes())},
		{Method: http.MethodGet, Pattern: "/v1/records/executions", Handler: gateway.NewExecutionRecordsHandler(recordsModule)},
		{Method: http.MethodGet, Pattern: "/v1/records/executions/{execution_id}", Handler: gateway.NewExecutionRecordHandler(recordsModule)},
		{Method: http.MethodGet, Pattern: "/v1/generations", Handler: gateway.NewGenerationsHandler(generationService)},
		{Method: http.MethodGet, Pattern: "/v1/generations/{execution_id}", Handler: gateway.NewGenerationHandler(generationService)},
	}
	allRoutes := make([]server.Route, 0, len(routes)+32)
	allRoutes = append(allRoutes, routes...)
	allRoutes = append(allRoutes, uiAuthRoutes(uiAuth, logger)...)
	allRoutes = append(allRoutes, runtimeRoutes(cfg, recordsModule, readiness)...)
	allRoutes = append(allRoutes, operationalEventsRoutes(eventHub, readiness, providerManager.Statuses, logger)...)
	allRoutes = append(allRoutes, providerAdminRoutes(providerAdmin)...)
	allRoutes = append(allRoutes, resourceAdminRoutes(resourceAdmin)...)
	allRoutes = append(allRoutes, concurrencyAdminRoutes(concurrencyAdmin)...)
	allRoutes = append(allRoutes, analyticsAdminRoutes(analyticsAdmin)...)

	router := server.NewRouter(logger, policy, health.Live(), readiness.Ready(), allRoutes...)
	return router, nil
}
