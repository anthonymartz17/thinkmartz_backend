package post

import (
	"context"

	"github.com/google/uuid"
)

// Service defines methods a concrete implementation must provide to be used by Handler.
type Service interface {
	Create(ctx context.Context, userID uuid.UUID, content string) (*Post, error)
}
