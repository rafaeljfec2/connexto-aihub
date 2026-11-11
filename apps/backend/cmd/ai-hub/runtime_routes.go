package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/records"
	"aihub/internal/server"
)

func runtimeRoutes(cfg config.Config, recordsModule *records.Service, readiness *health.Registry) []server.Route {
	envName := os.Getenv("AIHUB_RUNTIME_ENVIRONMENT")
	if envName == "" {
		envName = "development"
	}
	dataOrigin := os.Getenv("AIHUB_RUNTIME_DATA_ORIGIN")
	if dataOrigin == "" {
		dataOrigin = "live"
	}
	version := os.Getenv("AIHUB_RUNTIME_VERSION")
	if version == "" {
		version = "PREVIEW 01"
	}

	metaProvider := func() gateway.RuntimeMeta {
		status := "operational"
		if readiness != nil {
			report := readiness.Report(context.Background())
			if report.Status != health.StatusUp {
				status = "degraded"
			}
		}
		return gateway.RuntimeMeta{
			Environment:   envName,
			DataOrigin:    dataOrigin,
			Version:       version,
			GatewayStatus: status,
		}
	}

	return []server.Route{
		{
			Method:  http.MethodGet,
			Pattern: "/api/runtime/meta",
			Handler: gateway.NewRuntimeMetaHandler(metaProvider),
		},
		{
			Method:  http.MethodGet,
			Pattern: "/api/telemetry/traffic-summary",
			Handler: gateway.NewTrafficSummaryHandler(recordsModule, time.Now),
		},
	}
}
