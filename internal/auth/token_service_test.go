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
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAccessToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		jwtConfig := config.JWTConfig{
			Secret: "test-secret",
			Expiry: 15 * time.Minute,
		}
		svc := auth.NewTokenService(jwtConfig, nil)

		userID := uuid.New()

		// act
		tokenString, err := svc.IssueAccessToken(userID)
		require.NoError(t, err)

		var claims auth.AccessTokenClaims

		parsedToken, err := jwt.ParseWithClaims(tokenString, &claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtConfig.Secret), nil
		})
		require.NoError(t, err)

		// assert
		assert.True(t, parsedToken.Valid)
		assert.Equal(t, userID, claims.UserID)
		assert.WithinDuration(t, time.Now().Add(jwtConfig.Expiry), claims.ExpiresAt.Time, time.Second)
	})
}

func TestIssueRefreshToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		tokenService, rdb := newTestTokenService(t)
		userID := uuid.New()

		// act
		gotRefreshToken, gotErr := tokenService.IssueRefreshToken(t.Context(), userID)

		// assert
		require.NoError(t, gotErr, "should not fail to issueRefreshToken on success case")
		key := fmt.Sprintf("session:refresh_token:%s", gotRefreshToken)
		storedUserID, err := rdb.Get(ctx, key).Result()

		assert.NoError(t, err, "should not fail on success case")
		assert.Equal(t, userID.String(), storedUserID, "stored value should be the issuing user's ID")
	})

	t.Run("Fails when redis is unreachable", func(t *testing.T) {
		// arrange
		tokenService, rdb := newTestTokenService(t)
		userID := uuid.New()

		require.NoError(t, rdb.Close(), "should close redis client cleanly for setup")

		// act
		gotRefreshToken, gotErr := tokenService.IssueRefreshToken(t.Context(), userID)

		// assert
		assert.Error(t, gotErr, "should return an error when redis is unreachable")
		assert.Empty(t, gotRefreshToken, "should not return a token when redis write fails")
	})
}

func TestValidateRefreshToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		tokenService, _ := newTestTokenService(t)
		userID := uuid.New()

		opaque, err := tokenService.IssueRefreshToken(ctx, userID)
		require.NoError(t, err, "should not fail to set up issued refresh token")

		// act
		gotUserID, gotErr := tokenService.ValidateRefreshToken(ctx, opaque)

		// assert
		assert.NoError(t, gotErr, "should not fail on success case")
		assert.Equal(t, userID, gotUserID, "should return the issuing user's ID")
	})

	t.Run("Fails when token is not found", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		tokenService, _ := newTestTokenService(t)

		// act
		gotUserID, gotErr := tokenService.ValidateRefreshToken(ctx, "non-existent-opaque-token")

		// assert
		assert.Equal(t, uuid.Nil, gotUserID, "should return uuid.Nil when token is not found")
		assert.ErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound)
	})

	t.Run("Fails when stored value is malformed", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		tokenService, rdb := newTestTokenService(t)
		opaque := "malformed-value-opaque-token"
		key := fmt.Sprintf("session:refresh_token:%s", opaque)

		require.NoError(t, rdb.Set(ctx, key, "not-a-valid-uuid", time.Minute).Err(),
			"should set up a malformed stored value")

		// act
		gotUserID, gotErr := tokenService.ValidateRefreshToken(ctx, opaque)

		// assert
		assert.Equal(t, uuid.Nil, gotUserID)
		assert.Error(t, gotErr, "should return an error when the stored value is not a valid UUID")
		assert.NotErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound, "malformed data should not be mistaken for not-found")
	})

	t.Run("Fails when redis is unreachable", func(t *testing.T) {
		// arrange
		tokenService, rdb := newTestTokenService(t)
		require.NoError(t, rdb.Close(), "should close redis client cleanly for setup")

		// act
		gotUserID, gotErr := tokenService.ValidateRefreshToken(t.Context(), "any-opaque-token")

		// assert
		assert.Equal(t, uuid.Nil, gotUserID)
		assert.Error(t, gotErr, "should return an error when redis is unreachable")
		assert.NotErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound, "connection failure should not be reported as not-found")
	})
}

func TestInvalidateRefreshToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		tokenService, rdb := newTestTokenService(t)
		userID := uuid.New()

		opaque, err := tokenService.IssueRefreshToken(ctx, userID)
		require.NoError(t, err, "should not fail to set up issued refresh token")

		// act
		gotErr := tokenService.InvalidateRefreshToken(ctx, opaque)

		// assert
		assert.NoError(t, gotErr, "should not fail on success case")

		key := fmt.Sprintf("session:refresh_token:%s", opaque)
		_, err = rdb.Get(ctx, key).Result()
		assert.ErrorIs(t, err, redis.Nil, "token should no longer exist in redis after invalidation")
	})

	t.Run("Fails when token is not found", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		tokenService, _ := newTestTokenService(t)

		// act
		gotErr := tokenService.InvalidateRefreshToken(ctx, "non-existent-opaque-token")

		// assert
		assert.ErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound)
	})

	t.Run("Fails when redis is unreachable", func(t *testing.T) {
		// arrange
		tokenService, rdb := newTestTokenService(t)
		require.NoError(t, rdb.Close(), "should close redis client cleanly for setup")

		// act
		gotErr := tokenService.InvalidateRefreshToken(t.Context(), "any-opaque-token")

		// assert
		assert.Error(t, gotErr, "should return an error when redis is unreachable")
		assert.NotErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound, "connection failure should not be reported as not-found")
	})
}

func newTestTokenService(t *testing.T) (*auth.TokenService, *redis.Client) {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "config should load on setup")

	client := redisClient.NewRedisClient(cfg.Redis)
	tokenService := auth.NewTokenService(cfg.JWT, client)

	return tokenService, client
}
