package auth

import (
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAccessToken_Success(t *testing.T) {

	// arrange
	svc := &TokenService{
		JWTConfig: config.JWTConfig{
			Secret: "test-secret",
			Expiry: 15 * time.Minute,
		},
	}

	userID := uuid.New()

	// act
	tokenString, err := svc.IssueAccessToken(userID)
	require.NoError(t, err)

	var claims AccessTokenClaims

	parsedToken, err := jwt.ParseWithClaims(tokenString, &claims, func(_ *jwt.Token) (interface{}, error) {
		return []byte(svc.JWTConfig.Secret), nil
	})
	require.NoError(t, err)

	// assert
	assert.True(t, parsedToken.Valid)
	assert.Equal(t, userID, claims.UserID)
	assert.WithinDuration(t, time.Now().Add(svc.JWTConfig.Expiry), claims.ExpiresAt.Time, time.Second)
}
