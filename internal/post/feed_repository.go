package post

import (
	"context"

	"github.com/google/uuid"
)

// FeedRepository defines the methods a concrete repository must implement to
// fan a post out to its followers' feeds.
type FeedRepository interface {
	AddToFeed(ctx context.Context, p Post, followerIDs []uuid.UUID) error
}
