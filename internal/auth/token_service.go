package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTokenClaims defines the expected claims for the access token
type AccessTokenClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// validates TokenService implements TokenIssuer
var _ TokenIssuer = (*TokenService)(nil)

// TokenService handles pure JWT access-token signing and opaque
// refresh-token generation. It holds no storage — see SessionRepository
// for refresh-token session storage.
type TokenService struct {
	jwtConfig config.JWTConfig
}

// NewTokenService builds and returns a new TokenService
func NewTokenService(cfg config.JWTConfig) *TokenService {
	return &TokenService{
		jwtConfig: cfg,
	}
}

// IssueAccessToken generates, signs and return a new access token
func (t *TokenService) IssueAccessToken(userID uuid.UUID) (string, error) {
	claims := &AccessTokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.jwtConfig.Expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(t.jwtConfig.Secret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken generates a new random opaque refresh-token value.
// It does not store it — callers persist it via SessionRepository.
func (t *TokenService) GenerateRefreshToken() (string, error) {
	return generateRefreshTokenValue()
}

// generateRefreshTokenValue generates a random byte slice and encodes it to base64.URLEncoding
func generateRefreshTokenValue() (string, error) {
	b := make([]byte, 32) // 32 bytes = 256 bits of randomness
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
