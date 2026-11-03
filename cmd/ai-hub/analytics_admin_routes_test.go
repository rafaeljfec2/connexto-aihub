package main

import (
	"testing"
)

func TestAnalyticsAdminRoutesNilWhenDisabled(t *testing.T) {
	if routes := analyticsAdminRoutes(nil); len(routes) != 0 {
		t.Fatalf("len = %d, want 0", len(routes))
	}
}
