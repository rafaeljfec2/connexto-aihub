package main

import (
	"log/slog"
	"net/http"

	"aihub/internal/gateway"
	"aihub/internal/server"
	"aihub/internal/ui"
)

func uiAuthRoutes(gate ui.Gate, logger *slog.Logger) []server.Route {
	return []server.Route{
		{Method: http.MethodPost, Pattern: "/api/ui/login", Handler: gateway.NewUILoginHandler(gate, logger)},
		{Method: http.MethodPost, Pattern: "/api/ui/logout", Handler: gateway.NewUILogoutHandler(gate)},
		{Method: http.MethodGet, Pattern: "/api/ui/session", Handler: gateway.NewUISessionHandler(gate)},
	}
}
