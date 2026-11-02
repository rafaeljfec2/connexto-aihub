package execution

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// DefaultInactivityTimeout bounds silence between forwarded chunks.
const DefaultInactivityTimeout = 2 * time.Minute

// Manager registers, executes, and drains in-process Responses work.
type Manager struct {
	logger     *slog.Logger
	providerID string
	opener     OpenResponses
	inactivity time.Duration
	observer   ResultObserver

	mu            sync.Mutex
	accepting     bool
	active        map[string]*execution
	changed       chan struct{}
	cleanupFailed bool
}

// Option configures optional manager collaborators.
type Option func(*Manager)

// WithObserver installs the terminal-result observer. A nil observer is a
// no-op for isolated callers.
func WithObserver(observer ResultObserver) Option {
	return func(manager *Manager) {
		manager.observer = observer
	}
}

// NewManager builds the lifecycle owner for one configured provider adapter.
func NewManager(logger *slog.Logger, providerID string, opener OpenResponses, inactivity time.Duration, options ...Option) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	if inactivity <= 0 {
		inactivity = DefaultInactivityTimeout
	}

	manager := &Manager{
		logger:     logger,
		providerID: providerID,
		opener:     opener,
		inactivity: inactivity,
		accepting:  true,
		active:     make(map[string]*execution),
		changed:    make(chan struct{}),
	}
	for _, option := range options {
		if option != nil {
			option(manager)
		}
	}

	return manager
}

// Execute admits and runs one explicit command.
func (m *Manager) Execute(ctx context.Context, command Command, sink Sink) (Result, error) {
	if err := m.validate(command, sink); err != nil {
		return Result{}, err
	}
	if ctx == nil {
		return Result{}, ErrInvalidCommand
	}

	m.mu.Lock()
	if !m.accepting {
		m.mu.Unlock()
		return Result{}, ErrUnavailable
	}
	executionID, err := newExecutionID()
	if err != nil {
		m.mu.Unlock()
		return Result{}, fmt.Errorf("create execution id: %w", err)
	}
	work := newExecution(m, executionID, command, ctx)
	m.active[executionID] = work
	m.notifyLocked()
	m.mu.Unlock()

	m.logStarted(work)
	return work.run(sink), nil
}

// BeginDrain atomically closes execution admission.
func (m *Manager) BeginDrain() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.accepting {
		m.accepting = false
		m.notifyLocked()
	}
}

// CancelActive requests shutdown cancellation for every registered execution.
func (m *Manager) CancelActive() {
	m.mu.Lock()
	active := make([]*execution, 0, len(m.active))
	for _, work := range m.active {
		active = append(active, work)
	}
	m.mu.Unlock()

	for _, work := range active {
		work.cancelForShutdown()
	}
}

// Wait blocks until every registered execution has published its result.
func (m *Manager) Wait(ctx context.Context) error {
	for {
		m.mu.Lock()
		if len(m.active) == 0 {
			cleanupFailed := m.cleanupFailed
			m.mu.Unlock()
			if cleanupFailed {
				return ErrCleanupUnconfirmed
			}
			return nil
		}
		changed := m.changed
		m.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}

// Drained reports whether all execution work and body cleanup were confirmed.
func (m *Manager) Drained() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.active) == 0 && !m.cleanupFailed
}

func (m *Manager) validate(command Command, sink Sink) error {
	if m.opener == nil || sink == nil {
		return ErrInvalidCommand
	}
	if command.Target.ProviderID == "" || (m.providerID != "" && command.Target.ProviderID != m.providerID) {
		return ErrInvalidCommand
	}
	if command.Target.ModelID != "" && strings.TrimSpace(command.Target.ModelID) == "" {
		return ErrInvalidCommand
	}
	if command.ProviderSessionID == "" || command.ContentType == "" {
		return ErrInvalidCommand
	}
	return nil
}

func (m *Manager) remove(work *execution) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, work.id)
	m.notifyLocked()
}

// observe isolates consumer failures from result publication and deregistration.
func (m *Manager) observe(work *execution, result Result) {
	if m.observer == nil {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			m.logger.Error("execution observer failed",
				"class", "observer_panic",
				"execution_id", result.ExecutionID,
			)
		}
	}()

	m.observer.OnExecutionFinished(FinishedExecution{
		Result:    result,
		RequestID: work.command.RequestID,
		Target:    work.command.Target,
		StartedAt: work.startedAt,
	})
}

func (m *Manager) recordCleanupFailure() {
	m.mu.Lock()
	m.cleanupFailed = true
	m.mu.Unlock()
}

func (m *Manager) notifyLocked() {
	close(m.changed)
	m.changed = make(chan struct{})
}

func (m *Manager) logStarted(work *execution) {
	m.logger.InfoContext(work.ctx, "execution started",
		"execution_id", work.id,
		"request_id", work.command.RequestID,
		"provider_id", work.command.Target.ProviderID,
		"model_id", work.command.Target.ModelID,
	)
}

func (m *Manager) logFinished(result Result, command Command) {
	level := slog.LevelInfo
	if result.Outcome == OutcomeFailed || result.Outcome == OutcomeTimedOut {
		level = slog.LevelWarn
	}
	m.logger.Log(context.Background(), level, "execution finished",
		"execution_id", result.ExecutionID,
		"request_id", command.RequestID,
		"provider_id", command.Target.ProviderID,
		"model_id", command.Target.ModelID,
		"outcome", result.Outcome,
		"phase", result.Phase,
		"cause", result.Cause,
		"upstream_status", result.UpstreamStatus,
		"duration", result.Duration,
	)
}

func newExecutionID() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(random[:]), nil
}

func isHeaderTimeout(err error) bool {
	var timeout responseHeaderTimeout
	return errors.As(err, &timeout) && timeout.IsResponseHeaderTimeout()
}
