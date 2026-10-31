package feed

import (
	"context"

	"github.com/google/uuid"
)

// Cache defines the method redis_repository must provide to implement Cache
type Cache interface {
	Get(ctx context.Context, userID uuid.UUID, limit int) ([]uuid.UUID, error)
	Set(ctx context.Context, userID uuid.UUID, feed []Entry) error
}
