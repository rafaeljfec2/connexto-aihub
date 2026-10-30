// Package app runs the lifecycle of the application modules.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Module is a component with an explicit startup and shutdown lifecycle.
// Start must be atomic: when it returns an error the module is considered not
// started and Stop will not be called for it.
type Module interface {
	Name() string
	Start(context.Context) error
	Stop(context.Context) error
}

// App starts modules in declaration order and stops them in reverse order.
type App struct {
	logger  *slog.Logger
	modules []Module
}

// New validates the module set and builds the application runner.
func New(logger *slog.Logger, modules ...Module) (*App, error) {
	names := make(map[string]struct{}, len(modules))

	for _, module := range modules {
		if module == nil {
			return nil, errors.New("app: module must not be nil")
		}

		name := module.Name()
		if name == "" {
			return nil, errors.New("app: module name must not be empty")
		}

		if _, exists := names[name]; exists {
			return nil, fmt.Errorf("app: duplicate module name %q", name)
		}

		names[name] = struct{}{}
	}

	return &App{logger: logger, modules: modules}, nil
}

// Start initializes modules in order. When a module fails, modules started so
// far are stopped in reverse order before the error is returned.
func (a *App) Start(ctx context.Context) error {
	started := make([]Module, 0, len(a.modules))

	for _, module := range a.modules {
		if err := module.Start(ctx); err != nil {
			a.logger.Error("module start failed", "module", module.Name(), "error", err)
			rollbackErr := a.stopModules(ctx, started)

			return errors.Join(fmt.Errorf("start module %q: %w", module.Name(), err), rollbackErr)
		}

		started = append(started, module)
		a.logger.Info("module started", "module", module.Name())
	}

	return nil
}

// Stop shuts down every module in reverse order, joining all failures.
func (a *App) Stop(ctx context.Context) error {
	return a.stopModules(ctx, a.modules)
}

func (a *App) stopModules(ctx context.Context, modules []Module) error {
	var errs []error

	for index := len(modules) - 1; index >= 0; index-- {
		module := modules[index]

		if err := module.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop module %q: %w", module.Name(), err))
			continue
		}

		a.logger.Info("module stopped", "module", module.Name())
	}

	return errors.Join(errs...)
}
