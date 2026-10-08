package post

import (
	"context"

	"github.com/google/uuid"
)

// Service defines methods a concrete implementation must provide to be used by Handler.
type Service interface {
	Create(ctx context.Context, userID uuid.UUID, content string) (*Post, error)
	GetByID(ctx context.Context, postID uuid.UUID) (*Post, error)
	Update(ctx context.Context, input UpdateInput) (*Post, error)
	Delete(ctx context.Context, input DeleteInput) error
}
