package main

import (
	"log/slog"
	"net/http"

	"aihub/internal/gateway"
	"aihub/internal/health"
	"aihub/internal/operational"
	"aihub/internal/providers"
	"aihub/internal/server"
)

func operationalEventsRoutes(hub *operational.Hub, readiness *health.Registry, statuses func() []providers.Status, logger *slog.Logger) []server.Route {
	if hub == nil || readiness == nil || statuses == nil {
		return nil
	}

	return []server.Route{{
		Method:  http.MethodGet,
		Pattern: "/api/events",
		Handler: gateway.NewOperationalEventsHandler(hub, readiness, statuses, logger),
	}}
}
