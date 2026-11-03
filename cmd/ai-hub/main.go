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
	"aihub/internal/db"
	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/observability"
	"aihub/internal/providers"
	"aihub/internal/records"
	"aihub/internal/resources"
	resourcespg "aihub/internal/resources/pgstore"
	"aihub/internal/server"
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
	providerRegistry := providers.NewRegistry()
	providerRegistry.Bind(providers.OpenCodeGoID, providerClient)
	providerRegistry.Bind(providers.OpenRouterID, openRouterClient)
	providerModule := providers.NewRegistryManager(logger, providerRegistry)
	recordsModule := records.New(records.Options{
		Dir:               filepath.Join(cfg.Catalog.DataDir, "records"),
		Logger:            logger,
		MaxAge:            cfg.Records.MaxAge,
		MaxCount:          cfg.Records.MaxCount,
		MaxStorageBytes:   cfg.Records.MaxStorageBytes,
		SegmentBytes:      cfg.Records.SegmentBytes,
		RetentionInterval: cfg.Records.RetentionInterval,
	})
	executionManager := execution.NewMultiProviderManager(logger, openResponsesFromRegistry(providerRegistry),
		gateway.StreamInactivityTimeout, execution.WithObserver(recordsModule))

	var catalogStore catalog.SnapshotStore = catalog.NewStore(cfg.Catalog.DataDir, logger)

	readiness := health.NewRegistry()

	// Database module (ADR-020): present only in postgres persistence mode.
	// It is declared between the providers and the catalog so startup
	// migrates and pings before the catalog loads its snapshot, and shutdown
	// closes the pool only after the catalog and records stop.
	var databaseModule app.Module

	var (
		configModule       app.Module
		providerAdmin      *gateway.ProviderAdmin
		resourceAdmin      *gateway.ResourceAdmin
		resourceAdmission  gateway.ResourceAdmitter
	)

	if cfg.Catalog.Persistence == config.EnvCatalogPersistencePostgres {
		pool, err := db.Open(db.Settings{
			URL:            cfg.Database.URL,
			MaxConns:       cfg.Database.MaxConns,
			MinConns:       cfg.Database.MinConns,
			ConnectTimeout: cfg.Database.ConnectTimeout,
		})
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}

		databaseModule = db.NewModule(pool, logger)
		catalogStore = catalog.NewPostgresStore(pool, logger)

		providerConfig, err := buildProviderConfiguration(cfg, pool, providerRegistry, logger)
		if err != nil {
			return err
		}

		configModule = providerConfig.module
		providerAdmin = gateway.NewProviderAdmin(providerConfig.service, logger)
		resourceService := resources.NewService(resourcespg.New(pool))
		resourceAdmission = resourceService
		resourceAdmin = gateway.NewResourceAdmin(resourceService, logger)

		if err := readiness.Register("database", db.PingCheck(pool, cfg.Database.ConnectTimeout)); err != nil {
			return fmt.Errorf("register database readiness check: %w", err)
		}

		logger.Info("catalog persistence configured", "mode", config.EnvCatalogPersistencePostgres)
	} else {
		logger.Info("catalog persistence configured", "mode", config.EnvCatalogPersistenceFile)
	}

	catalogModule := catalog.NewManager(logger, providerModule.Statuses, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: providerClient.FetchModelDescriptors,
		providers.OpenRouterID: openRouterClient.FetchModelDescriptors,
	}, catalogStore, catalog.Options{
		StaleAfter:  cfg.Catalog.StaleAfter,
		SyncTimeout: cfg.Catalog.SyncTimeout,
	})

	router, err := buildGatewayRouter(cfg, logger, providerRegistry, providerModule, executionManager,
		catalogModule, recordsModule, readiness, providerAdmin, resourceAdmin, resourceAdmission)
	if err != nil {
		return err
	}

	httpServer := server.New(cfg.Server, router, logger,
		server.WithNonLoopbackAllowed(cfg.Gateway.AllowedHostsSet && len(cfg.Gateway.AllowedHosts) > 0),
	)
	serviceLifecycle := lifecycle.NewExecutionLifecycle(httpServer, executionManager, cfg.Server.ShutdownTimeout)
	guardedProvider := lifecycle.NewGuardedProviderModule(providerModule, executionManager)

	modules := make([]app.Module, 0, 6)
	if databaseModule != nil {
		modules = append(modules, databaseModule)
	}
	if configModule != nil {
		modules = append(modules, configModule)
	}
	modules = append(modules, guardedProvider, catalogModule, recordsModule, serviceLifecycle)

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
