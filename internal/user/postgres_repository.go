package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	_foreignKeyViolation = "23503"
	_followeeFKey        = "follows_followee_id_fkey"
)

var _ Repository = (*PostgresRepository)(nil)

// PostgresRepository is the Postgres-backed Repository.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository returns a Repository backed by db.
func NewPostgresRepository(p *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: p}
}

// SaveFollow inserts the follow, skipping duplicates.
func (r *PostgresRepository) SaveFollow(ctx context.Context, f Follow) error {
	const query = `
		INSERT INTO follows (follower_id, followee_id)
		VALUES ($1, $2)
		ON CONFLICT (follower_id, followee_id) DO NOTHING`

	if _, err := r.pool.Exec(ctx, query, f.FollowerID, f.FolloweeID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == _foreignKeyViolation &&
			pgErr.ConstraintName == _followeeFKey {
			return ErrUserNotFound
		}
		return fmt.Errorf("save follow: %w", err)
	}
	return nil
}
