// Package middleware provides HTTP middleware for the transport/http layer.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware gates a route group behind a valid access token.
func AuthMiddleware(jwtConfig config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")

			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing or malformed authorization header", http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			accessToken := parts[1]

			claims := &auth.AccessTokenClaims{}

			token, err := jwt.ParseWithClaims(accessToken, claims, func(_ *jwt.Token) (any, error) {
				return []byte(jwtConfig.Secret), nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
			if err != nil || !token.Valid {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			userID := claims.UserID
			if userID == uuid.Nil {
				http.Error(w, "token missing user id claim", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user's ID, previously set by
// AuthMiddleware, from the request context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
