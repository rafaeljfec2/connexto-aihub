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

	"aihub/internal/app"
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

	readiness := health.NewRegistry()

	router := server.NewRouter(logger, health.Live(), readiness.Ready(),
		server.Route{
			Method:  http.MethodPost,
			Pattern: "/v1/responses",
			Handler: gateway.NewResponsesHandler(providerClient, logger, gateway.StreamInactivityTimeout),
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
	)

	httpServer := server.New(cfg.Server, router, logger)

	application, err := app.New(logger, providerModule, httpServer)
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
