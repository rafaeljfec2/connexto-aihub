package main

import (
	"fmt"
	"log/slog"

	"aihub/internal/analytics"
	analyticspg "aihub/internal/analytics/pgstore"
	"aihub/internal/app"
	"aihub/internal/catalog"
	"aihub/internal/concurrency"
	concurrencypg "aihub/internal/concurrency/pgstore"
	"aihub/internal/config"
	"aihub/internal/db"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/generations"
	generationspg "aihub/internal/generations/pgstore"
	"aihub/internal/health"
	"aihub/internal/metering"
	meteringpg "aihub/internal/metering/pgstore"
	"aihub/internal/operational"
	"aihub/internal/providers"
	"aihub/internal/records"
	"aihub/internal/resources"
	resourcespg "aihub/internal/resources/pgstore"
)

type persistenceRuntime struct {
	catalogStore      catalog.SnapshotStore
	databaseModule    app.Module
	configModule      app.Module
	providerService   *providers.Service
	providerAdmin     *gateway.ProviderAdmin
	resourceAdmin     *gateway.ResourceAdmin
	concurrencyAdmin  *gateway.ConcurrencyAdmin
	analyticsAdmin    *gateway.AnalyticsAdmin
	resourceAdmission gateway.ResourceAdmitter
	concurrencyGate   gateway.ConcurrencyGate
	usageMetering     *metering.Service
	generationService *generations.Service
	executionManager  *execution.Manager
	quotaService      *providers.QuotaService
}

func buildPersistenceRuntime(cfg config.Config, logger *slog.Logger, registry *providers.Registry,
	providerModule *providers.Manager, recordsModule *records.Service, readiness *health.Registry,
	eventHub *operational.Hub, catalogSources map[string]catalog.SourceFunc) (persistenceRuntime, error) {
	runtime := persistenceRuntime{
		catalogStore:      catalog.NewStore(cfg.Catalog.DataDir, logger),
		generationService: generations.NewService(recordsModule, nil, logger),
		executionManager: execution.NewMultiProviderManager(logger, openResponsesFromRegistry(registry),
			gateway.StreamInactivityTimeout, execution.WithObserver(recordsModule)),
	}
	if cfg.Catalog.Persistence != config.EnvCatalogPersistencePostgres {
		logger.Info("catalog persistence configured", "mode", config.EnvCatalogPersistenceFile)
		return runtime, nil
	}

	pool, err := db.Open(db.Settings{
		URL: cfg.Database.URL, MaxConns: cfg.Database.MaxConns,
		MinConns: cfg.Database.MinConns, ConnectTimeout: cfg.Database.ConnectTimeout,
	})
	if err != nil {
		return persistenceRuntime{}, fmt.Errorf("open database: %w", err)
	}
	runtime.databaseModule = db.NewModule(pool, logger)
	runtime.catalogStore = catalog.NewPostgresStore(pool, logger)

	providerConfig, err := buildProviderConfiguration(cfg, pool, registry, logger)
	if err != nil {
		return persistenceRuntime{}, err
	}
	runtime.configModule = providerConfig.module
	runtime.providerService = providerConfig.service
	runtime.providerAdmin = gateway.NewProviderAdmin(providerConfig.service, logger)
	runtime.providerAdmin.BindOperationalEvents(eventHub, providerModule.Statuses)
	openRouterQuotaAdapter := providers.NewOpenRouterQuotaAdapter(nil, cfg.Provider.AllowedEndpointHosts)
	openCodeQuotaAdapter := providers.NewOpenCodeQuotaAdapter(nil, cfg.Provider.AllowedEndpointHosts)
	antigravityQuotaAdapter := providers.NewAntigravityQuotaAdapter(nil, cfg.Provider.AllowedEndpointHosts)
	quotaService := providers.NewQuotaService(providerConfig.service, map[string]providers.QuotaAdapter{
		providers.TypeOpenRouter:  openRouterQuotaAdapter,
		providers.TypeOpenCodeGo:  openCodeQuotaAdapter,
		providers.TypeAntigravity: antigravityQuotaAdapter,
	}, logger)
	runtime.providerAdmin.BindQuotaService(quotaService)
	runtime.quotaService = quotaService
	usageAnalyticsService := providers.NewCursorUsageAnalyticsService(providerConfig.service, nil, logger)
	runtime.providerAdmin.BindCursorUsageAnalytics(usageAnalyticsService)
	resourceService := resources.NewService(resourcespg.New(pool))
	runtime.resourceAdmission = resourceService
	runtime.resourceAdmin = gateway.NewResourceAdmin(resourceService, logger)
	concurrencyService := concurrency.NewService(concurrencypg.New(pool), concurrency.WithLogger(logger))
	runtime.concurrencyGate = concurrencyService
	runtime.concurrencyAdmin = gateway.NewConcurrencyAdmin(concurrencyService, logger)
	analyticsService := analytics.NewService(analyticspg.New(pool))
	runtime.analyticsAdmin = gateway.NewAnalyticsAdmin(analyticsService, resourceService, concurrencyService, logger)
	runtime.usageMetering = metering.NewService(meteringpg.New(pool), metering.WithLogger(logger))
	runtime.generationService = generations.NewService(recordsModule, generationspg.New(pool), logger).
		WithRetention(cfg.Records.RetentionInterval)
	runtime.executionManager = execution.NewMultiProviderManager(logger, openResponsesFromRegistry(registry),
		gateway.StreamInactivityTimeout,
		execution.WithObserver(concurrency.NewReleaseObserver(concurrencyService, recordsModule)))

	if err := readiness.Register("database", db.PingCheck(pool, cfg.Database.ConnectTimeout)); err != nil {
		return persistenceRuntime{}, fmt.Errorf("register database readiness check: %w", err)
	}
	logger.Info("catalog persistence configured", "mode", config.EnvCatalogPersistencePostgres)
	return runtime, nil
}
