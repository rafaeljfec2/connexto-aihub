package main

import "testing"

func TestConcurrencyAdminRoutesNilWhenUnavailable(t *testing.T) {
	if routes := concurrencyAdminRoutes(nil); len(routes) != 0 {
		t.Fatalf("routes = %d, want none in file persistence mode", len(routes))
	}
}
