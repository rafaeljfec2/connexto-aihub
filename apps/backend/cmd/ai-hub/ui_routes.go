package main

import (
	"log/slog"
	"net/http"

	"aihub/internal/gateway"
	"aihub/internal/server"
)

func uiAuthRoutes(bundle UIAuthBundle, logger *slog.Logger) []server.Route {
	if bundle.UsersMode {
		return []server.Route{
			{Method: http.MethodPost, Pattern: "/api/ui/login", Handler: gateway.NewUIUsersLoginHandler(bundle.Users, logger)},
			{Method: http.MethodPost, Pattern: "/api/ui/signup", Handler: gateway.NewUIUsersSignUpHandler(bundle.Users, logger)},
			{Method: http.MethodPost, Pattern: "/api/ui/refresh", Handler: gateway.NewUIRefreshHandler(bundle.Users)},
			{Method: http.MethodPost, Pattern: "/api/ui/logout", Handler: gateway.NewUIUsersLogoutHandler(bundle.Users, bundle.UserGate)},
			{Method: http.MethodGet, Pattern: "/api/ui/session", Handler: gateway.NewUIUsersSessionHandler(bundle.UserGate)},
			{Method: http.MethodPost, Pattern: "/api/ui/forgot-password", Handler: gateway.NewUIForgotPasswordHandler(bundle.Users)},
			{Method: http.MethodPost, Pattern: "/api/ui/reset-password", Handler: gateway.NewUIResetPasswordHandler(bundle.Users)},
		}
	}

	return []server.Route{
		{Method: http.MethodPost, Pattern: "/api/ui/login", Handler: gateway.NewUILoginHandler(bundle.Operator, logger)},
		{Method: http.MethodPost, Pattern: "/api/ui/logout", Handler: gateway.NewUILogoutHandler(bundle.Operator)},
		{Method: http.MethodGet, Pattern: "/api/ui/session", Handler: gateway.NewUISessionHandler(bundle.Operator)},
	}
}
