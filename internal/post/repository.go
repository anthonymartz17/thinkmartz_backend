// Package post handles post creation and management.
package post

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Checks if  repository implements Repository
var _ Repository = (*repository)(nil)

// repository provides a connection pool and methods to interact with database
type repository struct {
	Pool *pgxpool.Pool
}

// NewRepository creates and returns a new repository
func NewRepository(p *pgxpool.Pool) Repository {
	return &repository{
		Pool: p,
	}
}

// Save inserts a new post and populates its generated ID and timestamps.
func (r *repository) Save(ctx context.Context, p *Post) error {

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
