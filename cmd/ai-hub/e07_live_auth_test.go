package main

import (
	"testing"
)

func TestE07UnauthenticatedGatewayRequestDoesNotReachProvider(t *testing.T) {
	hub, observer, err := e07StartAuthProbeHub(t)
	if err != nil {
		t.Fatal("isolated authentication probe hub could not start")
	}
	t.Cleanup(func() {
		if err := hub.stop(); err != nil {
			t.Error("isolated authentication probe hub did not stop")
		}
	})

	before := observer.Snapshot().Calls
	status, unchanged, requestID := e07UnauthenticatedProbe(hub, observer)
	if status != 401 {
		t.Fatalf("unauthenticated gateway status = %d, want 401", status)
	}
	if requestID == "" {
		t.Fatal("unauthenticated probe did not expose a correlation identifier for pre-E05 exclusion")
	}
	if !unchanged || observer.Snapshot().Calls != before {
		t.Fatal("unauthenticated request reached the provider transport")
	}
}
