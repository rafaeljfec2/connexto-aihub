package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Settings carries the validated database configuration. It mirrors
// internal/config without depending on that package.
type Settings struct {
	URL            string
	MaxConns       int
	MinConns       int
	ConnectTimeout time.Duration
}

// Open creates the connection pool. Connections are established on first use
// (migrations, ping, repositories) and bounded by the pool configuration. DSN
// parse failures are reported with the URL redacted (ADR-020).
func Open(settings Settings) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(settings.URL)
	if err != nil {
		return nil, SafeError(settings.URL, err)
	}

	config.MaxConns = int32(settings.MaxConns)
	config.MinConns = int32(settings.MinConns)
	config.ConnConfig.ConnectTimeout = settings.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, SafeError(settings.URL, err)
	}

	return pool, nil
}
