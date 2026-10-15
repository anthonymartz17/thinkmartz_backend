package post

import (
	"context"

	"github.com/google/uuid"
)

// FeedWriter defines the methods a fan-out target must implement.
type FeedWriter interface {
	AddToFeed(ctx context.Context, p Post, followerIDs []uuid.UUID) error
	RemoveFromFeeds(ctx context.Context, postID uuid.UUID, followerIDs []uuid.UUID) error
}
