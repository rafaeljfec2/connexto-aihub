// Command ai-hub starts the local AI Hub gateway.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"aihub/cmd/ai-hub/lifecycle"
	"aihub/internal/app"
	"aihub/internal/catalog"
	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/observability"
	"aihub/internal/operational"
	"aihub/internal/providers"
	"aihub/internal/records"
	"aihub/internal/server"
	"aihub/internal/static"
)

const version = "0.1.0-dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ai-hub: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	envPath := flag.String("env-file", defaultEnvFilePath(), "path to a dotenv file with environment overrides")
	flag.Parse()

	envPathExplicit := false
	flag.Visit(func(visited *flag.Flag) {
		if visited.Name == "env-file" {
			envPathExplicit = true
		}
	})

	envFileLoaded := false
	if err := config.LoadDotEnv(*envPath); err != nil {
		if envPathExplicit || !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else {
		envFileLoaded = true
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := observability.NewLogger(os.Stdout, cfg.Log.Level.SlogLevel(), string(cfg.Log.Format))
	logger.Info("starting ai hub", "version", version)
	if envFileLoaded {
		logger.Info("loaded environment file", "path", *envPath)
	}

	for _, name := range config.UnknownEnvVariables() {
		logger.Warn("unknown AIHUB_ environment variable ignored", "variable", name)
	}

	providerClient, err := providers.NewClient(cfg.Provider.OpenCodeGo, nil)
	if err != nil {
		return fmt.Errorf("build provider: %w", err)
	}
	openRouterClient, err := providers.NewOpenRouterClient(providers.Settings{BaseURL: providers.OpenRouterEndpoint}, nil)
	if err != nil {
		return fmt.Errorf("build OpenRouter provider: %w", err)
	}
	connextoProxyClient, err := providers.NewConnextoProxyClient(providers.Settings{
		BaseURL:       cfg.Provider.ConnextoProxy.BaseURL,
		ManagementKey: cfg.Provider.ConnextoProxy.GatewayKey,
	}, nil)
	if err != nil {
		return fmt.Errorf("build Connexto Proxy provider: %w", err)
	}
	providerRegistry := providers.NewRegistry()
	providerRegistry.Bind(providers.OpenCodeGoID, providerClient)
	providerRegistry.Bind(providers.OpenRouterID, openRouterClient)
	providerRegistry.Bind(providers.CursorID, connextoProxyClient)
	providerRegistry.Bind(providers.ConnextoProxyID, connextoProxyClient)
	providerModule := providers.NewRegistryManager(logger, providerRegistry)
	eventHub := operational.NewHub(logger)
	providerModule.SetStateChangeNotifier(func() {
		gateway.PublishProviderStatus(eventHub, providerModule.Statuses)
	})
	catalogSources := map[string]catalog.SourceFunc{
		providers.OpenCodeGoID:    providerClient.FetchModelDescriptors,
		providers.OpenRouterID:    openRouterClient.FetchModelDescriptors,
		providers.CursorID:        connextoProxyClient.FetchModelDescriptors,
		providers.ConnextoProxyID: connextoProxyClient.FetchModelDescriptors,
	}
	recordsModule := records.New(records.Options{
		Dir:               filepath.Join(cfg.Catalog.DataDir, "records"),
		Logger:            logger,
		MaxAge:            cfg.Records.MaxAge,
		MaxCount:          cfg.Records.MaxCount,
		MaxStorageBytes:   cfg.Records.MaxStorageBytes,
		SegmentBytes:      cfg.Records.SegmentBytes,
		RetentionInterval: cfg.Records.RetentionInterval,
	})
	readiness := health.NewRegistry()
	runtime, err := buildPersistenceRuntime(cfg, logger, providerRegistry, providerModule, recordsModule,
		readiness, eventHub, catalogSources)
	if err != nil {
		return err
	}

	catalogModule := catalog.NewManager(logger, providerModule.Statuses, catalogSources, runtime.catalogStore, catalog.Options{
		StaleAfter:  cfg.Catalog.StaleAfter,
		SyncTimeout: cfg.Catalog.SyncTimeout,
	})

	if runtime.providerAdmin != nil {
		runtime.providerAdmin.BindCatalogManager(catalogModule)
	}

	operatorGate, err := cfg.BuildOperatorGate()
	if err != nil {
		return fmt.Errorf("operator ui gate: %w", err)
	}

	router, err := buildGatewayRouterWithGenerations(cfg, logger, providerRegistry, providerModule,
		runtime.executionManager, catalogModule, recordsModule, runtime.generationService,
		readiness, runtime.providerAdmin, runtime.resourceAdmin, runtime.concurrencyAdmin,
		runtime.analyticsAdmin, runtime.resourceAdmission, runtime.concurrencyGate,
		runtime.usageMetering, eventHub, operatorGate)
	if err != nil {
		return err
	}

	if cfg.UI.ServeEmbedded {
		staticHandler, staticErr := static.Handler()
		if staticErr != nil {
			return fmt.Errorf("embedded frontend: %w", staticErr)
		}
		router = static.Wrap(router, staticHandler)
		logger.Info("embedded operator ui enabled")
	}

	httpServer := server.New(cfg.Server, router, logger,
		server.WithNonLoopbackAllowed(cfg.Gateway.AllowedHostsSet && len(cfg.Gateway.AllowedHosts) > 0),
	)
	serviceLifecycle := lifecycle.NewExecutionLifecycle(httpServer, runtime.executionManager, cfg.Server.ShutdownTimeout)
	guardedProvider := lifecycle.NewGuardedProviderModule(providerModule, runtime.executionManager)

	modules := make([]app.Module, 0, 7)
	if runtime.databaseModule != nil {
		modules = append(modules, runtime.databaseModule)
	}
	if runtime.configModule != nil {
		modules = append(modules, runtime.configModule)
	}
	modules = append(modules, guardedProvider, catalogModule, recordsModule,
		runtime.generationService, serviceLifecycle)

	application, err := app.New(logger, modules...)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := application.Start(ctx); err != nil {
		return err
	}

	logger.Info("ai hub started", "address", httpServer.Addr())

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-httpServer.Errors():
		logger.Error("http server failed", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := application.Stop(shutdownCtx); err != nil {
		return err
	}

	logger.Info("ai hub stopped")
	return nil
}
