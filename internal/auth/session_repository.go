package auth

import (
	"context"

	"github.com/google/uuid"
)

// SessionRepository defines the methods for storing, atomically retrieving
// and invalidating, and clearing refresh-token sessions.
type SessionRepository interface {
	Set(ctx context.Context, token string, userID uuid.UUID) error
	GetDel(ctx context.Context, token string) (uuid.UUID, error)
	Clear(ctx context.Context, token string) error
}
