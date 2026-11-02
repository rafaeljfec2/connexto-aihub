package providers

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"
)

var (
	ErrAdapterNotFound       = errors.New("provider adapter is not registered")
	ErrResponsesNotSupported = errors.New("provider adapter does not support Responses")
	ErrProviderUnavailable   = errors.New("provider is disabled or has no credential")
)

// Adapter is the runtime side of a configured provider: it receives the
// configuration changes that the administrative service persists.
type Adapter interface {
	Reconfigure(settings Settings) error
	Enabled() bool
}

// ResponsesAdapter is a configured provider runtime that can execute Responses.
type ResponsesAdapter interface {
	Adapter
	OpenTargetResponses(context.Context, ResponsesTarget, []byte, string, string) (*http.Response, error)
}

// ModelListAdapter exposes a provider's native model-list response.
type ModelListAdapter interface {
	Adapter
	ListModels(context.Context) (*http.Response, error)
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

// ProviderIDs returns the bound IDs in stable order.
func (r *Registry) ProviderIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.adapters))
	for id := range r.adapters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// OpenTargetResponses resolves and invokes only the runtime bound to provider ID.
func (r *Registry) OpenTargetResponses(ctx context.Context, target ResponsesTarget, payload []byte, contentType, sessionID string) (*http.Response, error) {
	adapter, ok := r.Resolve(target.ProviderID)
	if !ok {
		return nil, ErrAdapterNotFound
	}
	responses, ok := adapter.(ResponsesAdapter)
	if !ok {
		return nil, ErrResponsesNotSupported
	}
	if !adapter.Enabled() {
		return nil, ErrProviderUnavailable
	}
	return responses.OpenTargetResponses(ctx, target, payload, contentType, sessionID)
}
