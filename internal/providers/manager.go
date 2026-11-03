package providers

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

// State describes runtime health; StateDisabled denotes an unavailable runtime.
type State string

const (
	StateDisabled  State = "disabled"
	StateUnknown   State = "unknown"
	StateHealthy   State = "healthy"
	StateUnhealthy State = "unhealthy"
)

// Status reports execution availability and health for one provider without carrying secrets.
type Status struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	State   State  `json:"state"`
}

// Manager owns initial provider state and diagnostics for bound runtimes.
type Manager struct {
	logger   *slog.Logger
	registry *Registry
	mu       sync.RWMutex
	states   map[string]State
}

// NewManager preserves the single-provider composition used by existing callers.
func NewManager(logger *slog.Logger, client *Client) *Manager {
	registry := NewRegistry()
	registry.Bind(OpenCodeGoID, client)
	return NewRegistryManager(logger, registry)
}

// NewRegistryManager manages every runtime currently bound by provider ID.
func NewRegistryManager(logger *slog.Logger, registry *Registry) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	if registry == nil {
		registry = NewRegistry()
	}
	manager := &Manager{logger: logger, registry: registry, states: make(map[string]State)}
	registry.SetRuntimeChangeObserver(func(providerID string) {
		manager.setState(providerID, StateUnknown)
	})
	registry.setHealthCheckObserver(func(providerID string, err error) {
		state := StateHealthy
		switch {
		case errors.Is(err, context.Canceled):
			state = StateUnknown
		case err != nil:
			state = StateUnhealthy
		}
		manager.setState(providerID, state)
	})
	return manager
}

func (m *Manager) Name() string {
	return "providers"
}

// Start initializes provider state without making external connectivity requests.
func (m *Manager) Start(_ context.Context) error {
	for _, id := range m.registry.ProviderIDs() {
		adapter, ok := m.registry.Resolve(id)
		if !ok {
			continue
		}
		if !adapter.Enabled() {
			m.setState(id, StateDisabled)
			m.logger.Info("provider disabled or missing credential", "provider", id)
			continue
		}
		m.setState(id, StateUnknown)
		m.logger.Info("provider health check pending", "provider", id)
	}
	return nil
}

func (m *Manager) Stop(context.Context) error {
	for _, id := range m.registry.ProviderIDs() {
		adapter, ok := m.registry.Resolve(id)
		if !ok {
			continue
		}
		if closer, ok := adapter.(interface{ CloseIdle() }); ok {
			closer.CloseIdle()
		}
	}
	return nil
}

func (m *Manager) Statuses() []Status {
	ids := m.registry.ProviderIDs()
	statuses := make([]Status, 0, len(ids))
	for _, id := range ids {
		adapter, ok := m.registry.Resolve(id)
		if !ok {
			continue
		}
		enabled := adapter.Enabled()
		state := m.state(id)
		if !enabled {
			state = StateDisabled
		}
		statuses = append(statuses, Status{ID: id, Enabled: enabled, State: state})
	}
	return statuses
}

func (m *Manager) setState(providerID string, state State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[providerID] = state
}

func (m *Manager) state(providerID string) State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if state, ok := m.states[providerID]; ok {
		return state
	}
	return StateUnknown
}
