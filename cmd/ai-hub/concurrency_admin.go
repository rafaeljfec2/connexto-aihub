package main

import (
	"net/http"

	"aihub/internal/gateway"
	"aihub/internal/server"
)

func concurrencyAdminRoutes(admin *gateway.ConcurrencyAdmin) []server.Route {
	if admin == nil {
		return nil
	}
	return []server.Route{
		{Method: http.MethodGet, Pattern: "/api/concurrency-policies", Handler: admin.List()},
		{Method: http.MethodPut, Pattern: "/api/concurrency-policies", Handler: admin.Upsert()},
		{Method: http.MethodGet, Pattern: "/api/concurrency-usage", Handler: admin.Usage()},
		{Method: http.MethodPost, Pattern: "/api/concurrency-policies/{id}/enable", Handler: admin.Enable()},
		{Method: http.MethodPost, Pattern: "/api/concurrency-policies/{id}/disable", Handler: admin.Disable()},
	}
}
