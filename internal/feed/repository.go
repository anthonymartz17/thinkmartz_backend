package feed

import (
	"context"

	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/google/uuid"
)

// Repository defines the methods for reading post data from Postgres to
// rebuild a user's feed on a cache miss. Methods are added as hydration
// needs grow (e.g. ListPostsBefore).
type Repository interface {
	ListFolloweePostsBefore(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) ([]post.Post, error)
	ListNonCelebrityFeedEntries(ctx context.Context, userID uuid.UUID, threshold, limit int) ([]Entry, error)
	ListWithCelebrityPosts(ctx context.Context, userID uuid.UUID, ids []uuid.UUID, threshold, limit int) ([]post.Post, error)
}
