package feed

import (
	"context"

	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/google/uuid"
)

// Reader defines the methods a concrete feed service must implement to be used by Handler.
type Reader interface {
	GetFeed(ctx context.Context, userID uuid.UUID, cursor *Cursor) ([]post.Post, *Cursor, error)
}
