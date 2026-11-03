package routing_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"aihub/internal/providers"
	"aihub/internal/routing"
)

type snapshotTestAdapter struct {
	settings providers.Settings
}

func (a *snapshotTestAdapter) Reconfigure(settings providers.Settings) error {
	a.settings = settings
	return nil
}

func (a *snapshotTestAdapter) Enabled() bool {
	return a.settings.Enabled && a.settings.APIKey != ""
}

func (a *snapshotTestAdapter) OpenTargetResponses(context.Context, providers.ResponsesTarget, []byte, string, string) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
}

func TestCaptureSnapshotRecordsRuntimeFacts(t *testing.T) {
	registry := providers.NewRegistry()
	adapter := &snapshotTestAdapter{}
	registry.Bind(providers.OpenCodeGoID, adapter)
	if err := adapter.Reconfigure(providers.Settings{BaseURL: "https://example.test", APIKey: "key", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	snapshot := routing.CaptureSnapshot(routing.SnapshotInput{
		Admin: []routing.AdminProviderFacts{
			{ID: providers.OpenCodeGoID, Known: true, Enabled: true, HasCredential: true},
		},
		Status:   []providers.Status{{ID: providers.OpenCodeGoID, Enabled: true, State: providers.StateUnhealthy}},
		Registry: registry,
	})

	runtimeFacts := snapshot.Runtime[providers.OpenCodeGoID]
	if !runtimeFacts.ResponsesSupported || !runtimeFacts.RuntimeEnabled {
		t.Fatalf("runtime facts = %+v", runtimeFacts)
	}

	if runtimeFacts.Health != providers.StateUnhealthy {
		t.Fatalf("health = %q, want unhealthy context preserved", runtimeFacts.Health)
	}
}
