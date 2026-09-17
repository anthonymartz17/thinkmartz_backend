// Package post handles post creation and management.
package post

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Checks if  PostgresRepository implements Repository
var _ Repository = (*PostgresRepository)(nil)

// PostgresRepository provides a connection pool and methods to interact with database
type PostgresRepository struct {
	Pool *pgxpool.Pool
}

// NewPostgresRepository creates and returns a new PostgresRepository
func NewPostgresRepository(p *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		Pool: p,
	}
}

// Save inserts a new post and populates its generated ID and timestamps.
func (r *PostgresRepository) Save(ctx context.Context, p *Post) error {

	query := `
	INSERT INTO posts (user_id, content)
	VALUES ($1, $2)
	RETURNING id, created_at, updated_at
	`
	err := r.Pool.QueryRow(
		ctx,
		query,
		p.UserID,
		p.Content,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert post: %w", err)
	}

	return nil
}
