package main

import (
	"net/http"

	"aihub/internal/gateway"
	"aihub/internal/server"
)

func resourceAdminRoutes(admin *gateway.ResourceAdmin) []server.Route {
	if admin == nil {
		return nil
	}

	return []server.Route{
		{Method: http.MethodGet, Pattern: "/api/resource-policies", Handler: admin.List()},
		{Method: http.MethodPut, Pattern: "/api/resource-policies", Handler: admin.Upsert()},
		{Method: http.MethodGet, Pattern: "/api/resource-usage", Handler: admin.Usage()},
		{Method: http.MethodPost, Pattern: "/api/resource-policies/{id}/enable", Handler: admin.Enable()},
		{Method: http.MethodPost, Pattern: "/api/resource-policies/{id}/disable", Handler: admin.Disable()},
	}
}
