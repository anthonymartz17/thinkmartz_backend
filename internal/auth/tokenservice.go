package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrRefreshTokenNotFound is returned when no match for refresh token is not found on redis
	ErrRefreshTokenNotFound = errors.New("token not found")
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

// IssueRefreshToken generates a new opaque refresh token and stores it in
// Redis keyed by the opaque token itself, with the userID as the value —
// this allows a future refresh/rotate flow to look up the owning user
// directly from the token presented in the cookie. Unlike keying by userID,
// this does not invalidate any previously issued refresh token for the same
// user, so multiple sessions can hold valid refresh
// tokens concurrently.
func (t *TokenService) IssueRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {

	opaque, err := generateRefreshTokenValue()
	if err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}

	key := fmt.Sprintf("session:refresh_token:%s", opaque)
	err = t.RedisClient.Set(ctx, key, userID.String(), t.JWTConfig.RefreshExpiry).Err()
	if err != nil {
		return "", fmt.Errorf("set refreshToken on redis: %w", err)
	}

	return opaque, nil
}

// generateRefreshTokenValue generates a random byte slice and encodes it to base64.URLEncoding
func generateRefreshTokenValue() (string, error) {
	b := make([]byte, 32) // 32 bytes = 256 bits of randomness
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateRefreshToken fetches and returns refresh token from redis if exists otherwise returns ErrRefreshTokenNotFound
func (t *TokenService) ValidateRefreshToken(ctx context.Context, opaque string) (uuid.UUID, error) {

	key := fmt.Sprintf("session:refresh_token:%s", opaque)
	val, err := t.RedisClient.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		return uuid.Nil, ErrRefreshTokenNotFound
	}

	if err != nil {
		return uuid.Nil, fmt.Errorf("get refresh token: %w", err)
	}

	userID, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse stored user id: %w", err)
	}

	return userID, nil

}
