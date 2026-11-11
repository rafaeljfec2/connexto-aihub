package db

import (
	"context"
	"time"

	log "log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PingCheck returns a readiness check that pings the pool with a bounded
// timeout, used in postgres mode (ADR-020).
func PingCheck(pool *pgxpool.Pool, timeout time.Duration) func(context.Context) error {
	return func(ctx context.Context) error {
		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return pool.Ping(pingCtx)
	}
}

// Module implements app.Module on the pool: Start migrates then pings (fail
// fast on an explicitly configured database), Stop closes the pool.
type Module struct {
	pool   *pgxpool.Pool
	logger *log.Logger
}

// NewModule builds the database lifecycle module. The pool must be open but
// not necessarily connected.
func NewModule(pool *pgxpool.Pool, logger *log.Logger) *Module {
	return &Module{pool: pool, logger: logger}
}

// Name implements app.Module.
func (m *Module) Name() string {
	return "database"
}

// Start migrates the schema and pings, so startup fails closed on an
// unreachable database (ADR-020).
func (m *Module) Start(ctx context.Context) error {
	if m.pool == nil {
		return errf("database pool is required")
	}

	applied, current, err := Migrate(ctx, m.pool)
	if err != nil {
		return errf("migrate database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := m.pool.Ping(pingCtx); err != nil {
		return errf("ping database: %w", err)
	}

	if m.logger != nil {
		if len(applied) > 0 {
			m.logger.Info("database schema migrated",
				"applied_versions", applied, "schema_version", current)
		} else {
			m.logger.Info("database schema up to date", "schema_version", current)
		}
	}

	return nil
}

// Stop closes the pool, joining in-flight connections (bounded by the pool
// shutdown policy).
func (m *Module) Stop(_ context.Context) error {
	if m.pool != nil {
		m.pool.Close()
	}

	return nil
}
