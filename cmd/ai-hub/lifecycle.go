package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"aihub/internal/app"
	"aihub/internal/execution"
	"aihub/internal/server"
)

type executionLifecycle struct {
	httpServer     *server.Server
	manager        *execution.Manager
	cleanupTimeout time.Duration
}

func newExecutionLifecycle(httpServer *server.Server, manager *execution.Manager, cleanupTimeout time.Duration) *executionLifecycle {
	return &executionLifecycle{httpServer: httpServer, manager: manager, cleanupTimeout: cleanupTimeout}
}

func (l *executionLifecycle) Name() string {
	return "http-execution-lifecycle"
}

func (l *executionLifecycle) Start(ctx context.Context) error {
	return l.httpServer.Start(ctx)
}

func (l *executionLifecycle) Stop(ctx context.Context) error {
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

type guardedProviderModule struct {
	module  app.Module
	manager *execution.Manager
}

func newGuardedProviderModule(module app.Module, manager *execution.Manager) *guardedProviderModule {
	return &guardedProviderModule{module: module, manager: manager}
}

func (m *guardedProviderModule) Name() string {
	return m.module.Name()
}

func (m *guardedProviderModule) Start(ctx context.Context) error {
	return m.module.Start(ctx)
}

func (m *guardedProviderModule) Stop(ctx context.Context) error {
	if !m.manager.Drained() {
		return errors.New("provider shutdown skipped because execution cleanup is unconfirmed")
	}
	return m.module.Stop(ctx)
}
