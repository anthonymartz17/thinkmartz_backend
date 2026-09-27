// Package post handles post creation and management.
package post

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

// GetFollowersByID returns the IDs of every user following the given userID.
func (r *PostgresRepository) GetFollowersByID(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT follower_id
		FROM follows
		WHERE followee_id = $1
	`

	var followerIDs []uuid.UUID

	rows, err := r.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query followers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID

		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan follower id: %w", err)
		}

		followerIDs = append(followerIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate followers: %w", err)
	}

	return followerIDs, nil
}

// CountFollowers  returns the number of followers a user has by userID
func (r *PostgresRepository) CountFollowers(ctx context.Context, userID uuid.UUID) (int, error) {

	query := `
		SELECT COUNT(*)
		FROM follows 
		WHERE followee_id = $1
	`
	var count int
	if err := r.Pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count followers: %w", err)
	}
	return count, nil
}
