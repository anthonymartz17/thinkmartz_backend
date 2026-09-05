package auth

import (
	"context"

	"github.com/google/uuid"
)

// TokenIssuer defines the methods required a concrete token struct must provide to fullfill TokenService interface.
type TokenIssuer interface {
	IssueAccessToken(userID uuid.UUID) (string, error)
	IssueRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateRefreshToken(ctx context.Context, opaque string) (uuid.UUID, error)
	InvalidateRefreshToken(ctx context.Context, token string) error
}
