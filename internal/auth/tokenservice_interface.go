package auth

import (
	"context"

	"github.com/google/uuid"
)

// TokenService defines the methods required a concrete token struct must provide to fullfill TokenService interface.
type TokenService interface {
	IssueAccessToken(userID uuid.UUID) (string, error)
	IssueRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)
}
