// Command ai-hub starts the local AI Hub gateway.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"aihub/internal/access"
	"aihub/internal/app"
	"aihub/internal/catalog"
	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/observability"
	"aihub/internal/providers"
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
	envPath := flag.String("env-file", ".env", "path to a dotenv file with environment overrides")
	flag.Parse()

	envPathExplicit := false
	flag.Visit(func(visited *flag.Flag) {
		if visited.Name == "env-file" {
			envPathExplicit = true
		}
	})

	if err := config.LoadDotEnv(*envPath); err != nil && (envPathExplicit || !errors.Is(err, os.ErrNotExist)) {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := observability.NewLogger(os.Stdout, cfg.Log.Level.SlogLevel(), string(cfg.Log.Format))
	logger.Info("starting ai hub", "version", version)

	for _, name := range config.UnknownEnvVariables() {
		logger.Warn("unknown AIHUB_ environment variable ignored", "variable", name)
	}

	providerClient, err := providers.NewClient(cfg.Provider.OpenCodeGo, nil)
	if err != nil {
		return fmt.Errorf("build provider: %w", err)
	}

	providerModule := providers.NewManager(logger, providerClient)

	catalogStore := catalog.NewStore(cfg.Catalog.DataDir, logger)
	catalogModule := catalog.NewManager(logger, providerModule.Statuses, map[string]catalog.SourceFunc{
		providers.OpenCodeGoID: providerClient.FetchModelDescriptors,
	}, catalogStore, catalog.Options{
		StaleAfter:  cfg.Catalog.StaleAfter,
		SyncTimeout: cfg.Catalog.SyncTimeout,
	})

	readiness := health.NewRegistry()

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey:               cfg.Gateway.APIKey,
		AllowedHosts:         cfg.Gateway.AllowedHosts,
		AllowedHostsExplicit: cfg.Gateway.AllowedHostsSet,
		AllowedOrigins:       cfg.Gateway.AllowedOrigins,
		MaxBodyBytes:         cfg.Gateway.MaxBodyBytes,
	})
	if err != nil {
		return fmt.Errorf("build gateway access policy: %w", err)
	}

	router := server.NewRouter(logger, policy, health.Live(), readiness.Ready(),
		server.Route{
			Method:    http.MethodPost,
			Pattern:   "/v1/responses",
			Handler:   gateway.NewResponsesHandler(providerClient, logger, gateway.StreamInactivityTimeout),
			Admission: access.ResponsesAdmission(policy.MaxBodyBytes()),
		},
		server.Route{
			Method:  http.MethodGet,
			Pattern: "/v1/models",
			Handler: gateway.NewModelsHandler(providerClient, logger),
		},
		server.Route{
			Method:  http.MethodGet,
			Pattern: "/diagnostics/providers",
			Handler: gateway.NewProvidersHandler(providerModule.Statuses),
		},
		server.Route{
			Method:  http.MethodGet,
			Pattern: "/v1/catalog/models",
			Handler: gateway.NewCatalogModelsHandler(catalogModule),
		},
		server.Route{
			Method:  http.MethodGet,
			Pattern: "/v1/catalog/models/{provider}/{model...}",
			Handler: gateway.NewCatalogModelHandler(catalogModule),
		},
		server.Route{
			Method:  http.MethodGet,
			Pattern: "/v1/catalog/providers",
			Handler: gateway.NewCatalogProvidersHandler(catalogModule),
		},
		server.Route{
			Method:    http.MethodPost,
			Pattern:   "/v1/catalog/sync",
			Handler:   gateway.NewCatalogSyncHandler(catalogModule),
			Admission: access.CatalogSyncAdmission(policy.MaxBodyBytes()),
		},
	)

	httpServer := server.New(cfg.Server, router, logger,
		server.WithNonLoopbackAllowed(cfg.Gateway.AllowedHostsSet && len(cfg.Gateway.AllowedHosts) > 0),
	)

	application, err := app.New(logger, providerModule, catalogModule, httpServer)
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
