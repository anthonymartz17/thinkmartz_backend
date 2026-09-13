// Package post handles post creation and management.
package post

import "github.com/jackc/pgx/v5/pgxpool"

// Repository provides a connection pool and methods to interact with database
type Repository struct {
	Pool *pgxpool.Pool
}

// NewRepository creates and returns a new Repository
func NewRepository(p *pgxpool.Pool) *Repository {
	return &Repository{
		Pool: p,
	}
}
