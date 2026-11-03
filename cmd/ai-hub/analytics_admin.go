package main

import (
	"net/http"

	"aihub/internal/gateway"
	"aihub/internal/server"
)

func analyticsAdminRoutes(admin *gateway.AnalyticsAdmin) []server.Route {
	if admin == nil {
		return nil
	}
	return []server.Route{
		{Method: http.MethodGet, Pattern: "/api/resource-analytics/admissions", Handler: admin.Admissions()},
		{Method: http.MethodGet, Pattern: "/api/resource-analytics/admissions/distribution", Handler: admin.Distribution()},
		{Method: http.MethodGet, Pattern: "/api/resource-analytics/quota-utilization", Handler: admin.QuotaUtilization()},
		{Method: http.MethodGet, Pattern: "/api/resource-analytics/concurrency", Handler: admin.Concurrency()},
	}
}
