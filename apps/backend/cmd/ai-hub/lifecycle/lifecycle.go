package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"aihub/internal/app"
	"aihub/internal/execution"
	"aihub/internal/server"
)

type ExecutionLifecycle struct {
	httpServer     *server.Server
	manager        *execution.Manager
	cleanupTimeout time.Duration
}

func NewExecutionLifecycle(httpServer *server.Server, manager *execution.Manager, cleanupTimeout time.Duration) *ExecutionLifecycle {
	return &ExecutionLifecycle{httpServer: httpServer, manager: manager, cleanupTimeout: cleanupTimeout}
}

func (l *ExecutionLifecycle) Name() string {
	return "http-execution-lifecycle"
}

func (l *ExecutionLifecycle) Start(ctx context.Context) error {
	return l.httpServer.Start(ctx)
}

func (l *ExecutionLifecycle) Stop(ctx context.Context) error {
	l.manager.BeginDrain()
	shutdownErr := l.httpServer.Stop(ctx)
	if shutdownErr == nil {
		if err := l.manager.Wait(ctx); err == nil {
			return nil
		} else {
			shutdownErr = fmt.Errorf("wait for executions during drain: %w", err)
		}
	}

	l.manager.CancelActive()
	closeErr := l.httpServer.ForceClose()
	cleanupCtx, cancel := context.WithTimeout(context.Background(), l.cleanupTimeout)
	defer cancel()
	cleanupErr := l.manager.Wait(cleanupCtx)
	if cleanupErr != nil {
		cleanupErr = fmt.Errorf("join execution cleanup: %w", cleanupErr)
	}
	return errors.Join(shutdownErr, closeErr, cleanupErr)
}

type GuardedProviderModule struct {
	module  app.Module
	manager *execution.Manager
}

func NewGuardedProviderModule(module app.Module, manager *execution.Manager) *GuardedProviderModule {
	return &GuardedProviderModule{module: module, manager: manager}
}

func (m *GuardedProviderModule) Name() string {
	return m.module.Name()
}

func (m *GuardedProviderModule) Start(ctx context.Context) error {
	return m.module.Start(ctx)
}

func (m *GuardedProviderModule) Stop(ctx context.Context) error {
	if !m.manager.Drained() {
		return errors.New("provider shutdown skipped because execution cleanup is unconfirmed")
	}
	return m.module.Stop(ctx)
}
