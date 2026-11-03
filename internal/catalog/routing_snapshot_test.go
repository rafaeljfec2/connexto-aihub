package catalog

import (
	"context"
	"testing"
	"time"

	"aihub/internal/providers"
)

func TestCaptureRoutingSnapshotSyncMatchesViews(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, func(context.Context) ([]providers.ModelDescriptor, error) {
		return []providers.ModelDescriptor{managerDescriptor("model-a", false, true)}, nil
	}, providers.StateHealthy, managerOptions(now))

	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	waitForOutcome(t, manager, SyncOutcomeSuccess)

	bundle := manager.CaptureRoutingSnapshot(map[string]providers.State{
		providers.OpenCodeGoID: providers.StateHealthy,
	})

	if len(bundle.Views) != 1 {
		t.Fatalf("views = %d, want 1", len(bundle.Views))
	}

	sync, ok := bundle.Sync[providers.OpenCodeGoID]
	if !ok || sync.LastSuccessAt == nil {
		t.Fatal("expected sync metadata in bundle")
	}

	if bundle.Views[0].Freshness != FreshnessFresh {
		t.Fatalf("freshness = %q, want fresh", bundle.Views[0].Freshness)
	}
}
