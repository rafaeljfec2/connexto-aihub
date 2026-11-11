package pgstore

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store implements concurrency.Repository against PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a PostgreSQL-backed concurrency store.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
