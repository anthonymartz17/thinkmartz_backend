// Package post handles post creation and management.
package post

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const _postLikesPostIDFKey = "post_likes_post_id_fkey" // confirm against the migration

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

// GetFollowerCount returns the user's follower count, maintained by a
// trigger on the follows table rather than computed on read.
func (r *PostgresRepository) GetFollowerCount(ctx context.Context, userID uuid.UUID) (int, error) {

	query := `
		SELECT follower_count
		FROM users
		WHERE id = $1
	`
	var count int
	if err := r.Pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("get follower count: %w", err)
	}
	return count, nil
}

// GetByID returns a single post that matches the given userID.
func (r *PostgresRepository) GetByID(ctx context.Context, postID uuid.UUID) (*Post, error) {
	query := `
		SELECT id, user_id, content, like_count, comment_count, created_at, updated_at
		FROM posts
		WHERE id = $1
	`

	var post Post

	err := r.Pool.QueryRow(ctx, query, postID).Scan(
		&post.ID,
		&post.UserID,
		&post.Content,
		&post.LikeCount,
		&post.CommentCount,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("get post by id: %w", err)

	}

	return &post, nil

}

// Update sets the content of the post with p.ID and populates p with the stored row.
func (r *PostgresRepository) Update(ctx context.Context, p *Post) error {
	query := `
		UPDATE posts
		SET content = $1, updated_at = now()
		WHERE id = $2
		RETURNING user_id, content,like_count,comment_count, created_at, updated_at 
	`

	err := r.Pool.QueryRow(ctx, query, p.Content, p.ID).Scan(
		&p.UserID,
		&p.Content,
		&p.LikeCount,
		&p.CommentCount,
		&p.CreatedAt,
		&p.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPostNotFound
		}
		return fmt.Errorf("update post: %w", err)
	}

	return nil
}

// Delete removes a post by ID.
// Returns ErrPostNotFound if no row matched.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM posts WHERE id = $1`

	tag, err := r.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrPostNotFound
	}

	return nil
}

// SaveLike records a user liking a post, idempotently. Returns ErrPostNotFound if the post doesn't exist.
func (r *PostgresRepository) SaveLike(ctx context.Context, like Like) error {
	const query = `
		INSERT INTO post_likes (user_id, post_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, post_id) DO NOTHING`

	_, err := r.Pool.Exec(ctx, query, like.UserID, like.PostID)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == pgerrcode.ForeignKeyViolation &&
		pgErr.ConstraintName == _postLikesPostIDFKey {
		return ErrPostNotFound
	}

	return fmt.Errorf("save like: %w", err)
}

// DeleteLike removes the user's like on a post. Removing a like that
// doesn't exist is a no-op: the end state ("not liked") is already true.
func (r *PostgresRepository) DeleteLike(ctx context.Context, like Like) error {
	const query = `
		DELETE FROM post_likes
		WHERE user_id = $1 AND post_id = $2`

	if _, err := r.Pool.Exec(ctx, query, like.UserID, like.PostID); err != nil {
		return fmt.Errorf("delete like: %w", err)
	}
	return nil
}
