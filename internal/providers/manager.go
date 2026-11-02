package providers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

// State is the availability state of a configured provider.
type State string

const (
	StateDisabled  State = "disabled"
	StateUnknown   State = "unknown"
	StateHealthy   State = "healthy"
	StateUnhealthy State = "unhealthy"
)

// Status is the diagnostics view of one provider; it never carries secrets.
type Status struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	State   State  `json:"state"`
}

// Manager owns startup probes and diagnostics for bound provider runtimes.
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
	return &Manager{logger: logger, registry: registry, states: make(map[string]State)}
}

func (m *Manager) Name() string {
	return "providers"
}

// Start probes each enabled runtime independently; provider failures do not fail startup.
func (m *Manager) Start(ctx context.Context) error {
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
		if err := verifyRuntime(ctx, adapter); err != nil {
			m.setState(id, StateUnhealthy)
			m.logger.Warn("provider connectivity check failed", "provider", id, "error", err)
			continue
		}
		m.setState(id, StateHealthy)
		m.logger.Info("provider connectivity verified", "provider", id)
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

func verifyRuntime(ctx context.Context, adapter Adapter) error {
	models, ok := adapter.(ModelListAdapter)
	if !ok {
		return fmt.Errorf("provider does not expose a model-list endpoint")
	}
	response, err := models.ListModels(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("connectivity check returned status %d", response.StatusCode)
	}
	return nil
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
