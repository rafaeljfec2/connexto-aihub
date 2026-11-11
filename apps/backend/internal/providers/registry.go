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
	mu                    sync.RWMutex
	adapters              map[string]Adapter
	runtimeVersions       map[string]uint64
	runtimeChangeObserver func(string)
	healthCheckObserver   func(string, error)
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters:        make(map[string]Adapter),
		runtimeVersions: make(map[string]uint64),
	}
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

// SetRuntimeChangeObserver registers the consumer that invalidates observed
// provider state after an adapter has been reconfigured.
func (r *Registry) SetRuntimeChangeObserver(observer func(string)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runtimeChangeObserver = observer
}

func (r *Registry) setHealthCheckObserver(observer func(string, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healthCheckObserver = observer
}

func (r *Registry) healthCheckVersion(providerID string) uint64 {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.runtimeVersions[providerID]
}

func (r *Registry) reportHealthCheck(providerID string, version uint64, err error) {
	if r == nil {
		return
	}
	r.mu.RLock()
	if r.runtimeVersions[providerID] != version {
		r.mu.RUnlock()
		return
	}
	observer := r.healthCheckObserver
	if observer != nil {
		// Keep the read lock through the state update so a reconfiguration
		// cannot be followed by a stale health result.
		observer(providerID, err)
	}
	r.mu.RUnlock()
}

// Reconfigure applies settings to the adapter bound to providerID. An
// unbound provider remains administrable without affecting another runtime.
func (r *Registry) Reconfigure(providerID string, settings Settings) error {
	adapter, ok := r.Resolve(providerID)
	if !ok {
		return nil
	}
	if err := adapter.Reconfigure(settings); err != nil {
		return err
	}

	r.mu.Lock()
	r.runtimeVersions[providerID]++
	observer := r.runtimeChangeObserver
	r.mu.Unlock()
	if observer != nil {
		observer(providerID)
	}
	return nil
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
