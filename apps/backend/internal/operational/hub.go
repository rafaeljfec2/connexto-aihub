package operational

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Event is one operational SSE payload ready to write on the wire.
type Event struct {
	Name string
	Data []byte
}

// Hub fan-outs operational events to active SSE subscribers.
type Hub struct {
	logger *slog.Logger
	mu     sync.RWMutex
	subs   map[chan Event]struct{}
}

// NewHub builds an operational event hub.
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}

	return &Hub{
		logger: logger,
		subs:   make(map[chan Event]struct{}),
	}
}

// Subscribe registers a buffered subscriber. The returned cancel function must be called.
func (h *Hub) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 64)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}

	return ch, cancel
}

// Publish marshals payload with schema_version and broadcasts to subscribers.
func (h *Hub) Publish(name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("operational event marshal failed", "event", name, "error", err.Error())
		return
	}

	event := Event{Name: name, Data: data}

	h.mu.RLock()
	snapshot := make([]chan Event, 0, len(h.subs))
	for ch := range h.subs {
		snapshot = append(snapshot, ch)
	}
	h.mu.RUnlock()

	for _, ch := range snapshot {
		select {
		case ch <- event:
		default:
			h.logger.Warn("operational subscriber slow; dropping event", "event", name)
		}
	}
}
