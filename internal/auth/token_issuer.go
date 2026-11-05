package auth

import "github.com/google/uuid"

// TokenIssuer defines the methods a concrete token issuer must provide:
// pure JWT access-token signing and opaque refresh-token generation, with
// no storage — see SessionRepository for refresh-token session storage.
type TokenIssuer interface {
	IssueAccessToken(userID uuid.UUID) (string, error)
	GenerateRefreshToken() (string, error)
}
