package providers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

// State is the availability state of a configured provider. Configuration
// (enabled/disabled) and availability (unknown/healthy/unhealthy) stay
// separate, so a missing credential can never flip application readiness.
type State string

// Provider states.
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

// Manager owns the provider lifecycle and its diagnostics state. It satisfies
// the app.Module contract and never fails startup because of the provider.
type Manager struct {
	logger *slog.Logger
	client *Client

	mu    sync.RWMutex
	state State
}

// NewManager builds the provider module.
func NewManager(logger *slog.Logger, client *Client) *Manager {
	return &Manager{logger: logger, client: client, state: StateUnknown}
}

// Name implements app.Module.
func (m *Manager) Name() string {
	return "providers"
}

// Start resolves the initial provider state. A disabled provider only logs;
// a failed connectivity check is recorded as unhealthy, never fatal.
func (m *Manager) Start(ctx context.Context) error {
	if !m.client.Enabled() {
		m.setState(StateDisabled)
		m.logger.Info("provider disabled; no credential configured", "provider", OpenCodeGoID)

		return nil
	}

	m.setState(StateUnknown)

	if err := m.verify(ctx); err != nil {
		m.setState(StateUnhealthy)
		m.logger.Warn("provider connectivity check failed", "provider", OpenCodeGoID, "error", err)

		return nil
	}

	m.setState(StateHealthy)
	m.logger.Info("provider connectivity verified", "provider", OpenCodeGoID)

	return nil
}

// Stop releases the upstream connection pool.
func (m *Manager) Stop(_ context.Context) error {
	m.client.CloseIdle()

	return nil
}

// Statuses snapshots the diagnostics state of every known provider.
func (m *Manager) Statuses() []Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return []Status{{
		ID:      OpenCodeGoID,
		Enabled: m.client.Enabled(),
		State:   m.state,
	}}
}

func (m *Manager) setState(state State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = state
}

func (m *Manager) verify(ctx context.Context) error {
	response, err := m.client.ListModels(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("connectivity check returned status %d", response.StatusCode)
	}

	return nil
}
