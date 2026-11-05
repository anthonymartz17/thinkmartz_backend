package auth_test

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
		svc := auth.NewTokenService(jwtConfig)

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

func TestGenerateRefreshToken(t *testing.T) {
	svc := auth.NewTokenService(config.JWTConfig{})

	t.Run("returns a non-empty, URL-safe base64 token", func(t *testing.T) {
		// act
		token, err := svc.GenerateRefreshToken()

		// assert
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		_, err = base64.URLEncoding.DecodeString(token)
		assert.NoError(t, err, "token should be valid URL-safe base64")
	})

	t.Run("returns distinct values across calls", func(t *testing.T) {
		// act
		first, err := svc.GenerateRefreshToken()
		require.NoError(t, err)

		second, err := svc.GenerateRefreshToken()
		require.NoError(t, err)

		// assert
		assert.NotEqual(t, first, second)
	})
}
