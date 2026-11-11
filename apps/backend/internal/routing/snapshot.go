package routing

import (
	"time"

	"aihub/internal/catalog"
	"aihub/internal/providers"
)

// SnapshotInput collects live facts for one coherent routing snapshot.
type SnapshotInput struct {
	Now        time.Time
	StaleAfter time.Duration
	Admin      []AdminProviderFacts
	Status     []providers.Status
	Registry   *providers.Registry
	Catalog    *catalog.Manager
}

// CaptureSnapshot builds a routing snapshot from the supplied sources.
func CaptureSnapshot(input SnapshotInput) Snapshot {
	capturedAt := input.Now
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}

	admin := make(map[string]AdminProviderFacts, len(input.Admin))
	for _, record := range input.Admin {
		admin[record.ID] = record
	}

	statuses := make(map[string]providers.Status, len(input.Status))
	for _, status := range input.Status {
		statuses[status.ID] = status
	}

	providerStates := make(map[string]providers.State, len(statuses))
	for id, status := range statuses {
		providerStates[id] = status.State
	}

	runtime := make(map[string]RuntimeProviderFacts)

	if input.Registry != nil {
		for _, id := range input.Registry.ProviderIDs() {
			adapter, bound := input.Registry.Resolve(id)
			facts := RuntimeProviderFacts{Bound: bound}
			if bound {
				_, responses := adapter.(providers.ResponsesAdapter)
				facts.ResponsesSupported = responses
				facts.RuntimeEnabled = adapter.Enabled()
			}

			if status, ok := statuses[id]; ok {
				facts.Health = status.State
			} else {
				facts.Health = providers.StateUnknown
			}

			runtime[id] = facts
		}
	}

	var models []catalog.View
	var sync map[string]catalog.ProviderSyncState

	if input.Catalog != nil {
		bundle := input.Catalog.CaptureRoutingSnapshot(providerStates)
		models = bundle.Views
		sync = bundle.Sync
	}

	return Snapshot{
		CapturedAt: capturedAt,
		StaleAfter: input.StaleAfter,
		Admin:      admin,
		Runtime:    runtime,
		Models:     models,
		Sync:       sync,
	}
}
