// Package pgstore persists E09 resource policies and admission counters in
// PostgreSQL. The domain package never imports pgx.
package pgstore

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store implements resources.Repository.
type Store struct {
	pool *pgxpool.Pool
}

// New builds a store on a migrated pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
