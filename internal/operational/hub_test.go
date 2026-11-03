package operational_test

import (
	"encoding/json"
	"sync"
	"testing"

	"aihub/internal/operational"
)

func TestHubPublishDeliversToSubscriber(t *testing.T) {
	hub := operational.NewHub(nil)
	events, cancel := hub.Subscribe()
	defer cancel()

	hub.Publish(operational.EventHeartbeat, map[string]int{"schema_version": operational.SchemaVersion})

	select {
	case event := <-events:
		if event.Name != operational.EventHeartbeat {
			t.Fatalf("event name = %q, want %q", event.Name, operational.EventHeartbeat)
		}
		var payload map[string]any
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload["schema_version"] != float64(operational.SchemaVersion) {
			t.Fatalf("schema_version = %v", payload["schema_version"])
		}
	default:
		t.Fatal("expected event on subscriber channel")
	}
}

func TestHubCancelDoesNotPanicConcurrentPublish(t *testing.T) {
	hub := operational.NewHub(nil)
	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers * 2)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			events, cancel := hub.Subscribe()
			_ = events
			cancel()
		}()
	}

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			hub.Publish(operational.EventProviderStatus, map[string]any{
				"schema_version": operational.SchemaVersion,
				"providers":        []any{},
			})
		}()
	}

	wg.Wait()
}
