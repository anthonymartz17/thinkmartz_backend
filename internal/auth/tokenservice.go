package auth

import (
	"fmt"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// AccessTokenClaims defines the expected claims for the access token
type AccessTokenClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// TokenService handles JWT and Opaque refresh token signing and issuance.
type TokenService struct {
	JWTConfig   config.JWTConfig
	RedisClient *redis.Client
}

// NewTokenService builds and returns a new TokenService
func NewTokenService(t config.JWTConfig, client *redis.Client) *TokenService {
	fmt.Print(client)
	return &TokenService{
		JWTConfig:   t,
		RedisClient: client,
	}

}

// IssueAccessToken generates, signs and return a new access token
func (t *TokenService) IssueAccessToken(userID uuid.UUID) (string, error) {
	claims := &AccessTokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.JWTConfig.Expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(t.JWTConfig.Secret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, nil
}
