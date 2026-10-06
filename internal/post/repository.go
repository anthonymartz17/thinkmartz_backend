package post

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the methods a concrete repository must implement to be used by Service.
type Repository interface {
	Save(ctx context.Context, p *Post) error
	GetFollowersByID(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	CountFollowers(ctx context.Context, userID uuid.UUID) (int, error)
	GetByID(ctx context.Context, postID uuid.UUID) (*Post, error)
	Update(ctx context.Context, p *Post) error
}
