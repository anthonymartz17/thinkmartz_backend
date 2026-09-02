package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	redisClient "github.com/anthonymartz17/thinkmartz_backend/internal/redis"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAccessToken_Success(t *testing.T) {

	// arrange
	svc := &auth.TokenService{
		JWTConfig: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 15 * time.Minute,
		},
	}

	userID := uuid.New()

	// act
	tokenString, err := svc.IssueAccessToken(userID)
	require.NoError(t, err)

	var claims auth.AccessTokenClaims

	parsedToken, err := jwt.ParseWithClaims(tokenString, &claims, func(_ *jwt.Token) (interface{}, error) {
		return []byte(svc.JWTConfig.Secret), nil
	})
	require.NoError(t, err)

	// assert
	assert.True(t, parsedToken.Valid)
	assert.Equal(t, userID, claims.UserID)
	assert.WithinDuration(t, time.Now().Add(svc.JWTConfig.Expiry), claims.ExpiresAt.Time, time.Second)
}

func TestIssueRefreshToken_Success(t *testing.T) {
	// Arrange
	ctx := t.Context()
	tokenService := newTestTokenService(t)
	userID := uuid.New()

	// act
	gotRefreshToken, gotErr := tokenService.IssueRefreshToken(t.Context(), userID)

	// assert
	require.NoError(t, gotErr, "should not fail to issueRefreshToken on success case")
	key := fmt.Sprintf("refresh_token:%s", gotRefreshToken)
	storedUserID, err := tokenService.RedisClient.Get(ctx, key).Result()

	assert.NoError(t, err, "should not fail on success case")
	assert.Equal(t, userID.String(), storedUserID, "stored value should be the issuing user's ID")

}

func TestIssueRefreshToken_RedisFailure(t *testing.T) {
	// Arrange
	tokenService := newTestTokenService(t)
	userID := uuid.New()

	require.NoError(t, tokenService.RedisClient.Close(), "should close redis client cleanly for setup")

	// act
	gotRefreshToken, gotErr := tokenService.IssueRefreshToken(t.Context(), userID)

	// assert
	assert.Error(t, gotErr, "should return an error when redis is unreachable")
	assert.Empty(t, gotRefreshToken, "should not return a token when redis write fails")
}

func newTestTokenService(t *testing.T) *auth.TokenService {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "config should load on setup")

	client := redisClient.NewRedisClient(cfg.Redis)
	tokenService := auth.NewTokenService(cfg.JWT, client)

	return tokenService
}
