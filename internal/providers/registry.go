package providers

import "sync"

// Adapter is the runtime side of a configured provider: it receives the
// configuration changes that the administrative service persists.
type Adapter interface {
	Reconfigure(settings Settings) error
	Enabled() bool
}

// Registry resolves the runtime adapter bound to a provider configuration.
// Bindings are keyed by provider id, not by adapter type: two configurations of
// the same type must not share one runtime, and only configurations with a
// binding receive traffic. The registry holds no business rules.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]Adapter)}
}

// Bind attaches the runtime adapter to a provider id.
func (r *Registry) Bind(providerID string, adapter Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.adapters[providerID] = adapter
}

// Resolve returns the adapter bound to a provider id.
func (r *Registry) Resolve(providerID string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[providerID]

	return adapter, ok
}
