package auth

import "github.com/anthonymartz17/thinkmartz_backend/internal/config"

// TokenService handles JWT and Opaque refresh token signing and issuance.
type TokenService struct {
	JWTConfig   config.JWTConfig
	RedisConfig config.RedisConfig
}

// NewTokenService builds and returns a new TokenService
func NewTokenService(t config.JWTConfig, r config.RedisConfig) *TokenService {
	return &TokenService{
		JWTConfig:   t,
		RedisConfig: r,
	}

}
