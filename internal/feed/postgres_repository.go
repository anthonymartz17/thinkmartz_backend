package feed

import (
	"context"
	"fmt"

	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository provides a connection pool and methods to interact with database
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates and returns a new PostgresRepository
func NewPostgresRepository(p *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: p,
	}
}

// ListFolloweePostsBefore returns up to limit posts by the users that userID
// follows, newest first, starting strictly after cursor in
// (created_at, id) order.
func (r *PostgresRepository) ListFolloweePostsBefore(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) ([]post.Post, error) {

	query := `
		SELECT 	
			p.id, 
			p.user_id, 
			p.content, 
			p.like_count, 
			p.comment_count,
			p.created_at, 
			p.updated_at
		FROM follows f
		JOIN posts p ON p.user_id = f.followee_id
		WHERE f.follower_id = $1
		AND (p.created_at, p.id) < ($2, $3)
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $4; 
	`

	rows, err := r.pool.Query(ctx, query, userID, cursor.CreatedAt, cursor.PostID, limit)
	if err != nil {
		return nil, fmt.Errorf("list followee posts before cursor: %w", err)
	}
	defer rows.Close()

	var posts []post.Post

	for rows.Next() {
		var p post.Post

		if err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.Content,
			&p.LikeCount,
			&p.CommentCount,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan followee post: %w", err)
		}

		posts = append(posts, p)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate followee posts: %w", err)
	}

	return posts, nil

}

// ListNonCelebrityFeedEntries returns up to limit feed entries (post ID and
// created_at) from the non-celebrity users that userID follows, newest first.
// Used to rebuild a feed on a cache miss.
func (r *PostgresRepository) ListNonCelebrityFeedEntries(ctx context.Context, userID uuid.UUID, threshold, limit int) ([]Entry, error) {
	query := `
		SELECT p.id ,p.created_at
		FROM follows f
		JOIN posts p ON p.user_id = f.followee_id 
		JOIN users u ON  u.id = f.followee_id
		WHERE f.follower_id = $1 AND u.follower_count < $2
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $3

	`

	rows, err := r.pool.Query(ctx, query, userID, threshold, limit)

	if err != nil {
		return nil, fmt.Errorf("query non-celebrity feed entries: %w", err)
	}
	defer rows.Close()

	var feed []Entry

	for rows.Next() {
		var e Entry

		if err := rows.Scan(&e.PostID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan non-celebrity feed rows: %w", err)
		}
		feed = append(feed, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate non-celebrity  feed rows: %w", err)
	}
	return feed, nil
}

// ListWithCelebrityPosts returns the posts with the given IDs merged with the
// newest posts from celebrities that userID follows (follower_count >=
// threshold), newest first, up to limit in total.
func (r *PostgresRepository) ListWithCelebrityPosts(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, threshold, limit int) ([]post.Post, error) {

	query := `
				(
			SELECT p.id, p.user_id, p.content, p.like_count,
						p.comment_count, p.created_at, p.updated_at
			FROM posts p
			WHERE p.id = ANY($2)
		)
		UNION ALL
		(
			SELECT p.id, p.user_id, p.content, p.like_count,
						p.comment_count, p.created_at, p.updated_at
			FROM follows f
			JOIN users u ON u.id = f.followee_id
			JOIN posts p ON p.user_id = f.followee_id
			WHERE f.follower_id = $1
				AND u.follower_count >= $3
			ORDER BY p.created_at DESC, p.id DESC
			LIMIT $4
		)
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`

	rows, err := r.pool.Query(ctx, query, userID, ids, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("list posts with celebrity posts: %w", err)
	}
	defer rows.Close()

	var posts []post.Post

	for rows.Next() {
		var p post.Post

		if err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.Content,
			&p.LikeCount,
			&p.CommentCount,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan post rows: %w", err)
		}

		posts = append(posts, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post rows: %w", err)
	}

	return posts, nil
}
