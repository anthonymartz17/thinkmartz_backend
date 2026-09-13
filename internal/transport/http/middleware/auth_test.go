package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	cfg := config.JWTConfig{Secret: "test-secret", Expiry: 15 * time.Minute}

	validClaims := func(userID uuid.UUID) *auth.AccessTokenClaims {
		return &auth.AccessTokenClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.Expiry)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
	}
	successUserID := uuid.New()

	tests := []struct {
		name           string
		authHeader     string
		wantStatus     int
		wantBody       string
		wantNextCalled bool
		wantUserID     uuid.UUID
	}{
		{
			name:       "missing authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "missing or malformed authorization header",
		},
		{
			name:       "malformed authorization header",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "missing or malformed authorization header",
		},
		{
			name:       "invalid token",
			authHeader: "Bearer not-a-real-jwt",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "invalid or expired token",
		},
		{
			name: "expired token",
			authHeader: "Bearer " + signToken(t, jwt.SigningMethodHS256, cfg.Secret,
				&auth.AccessTokenClaims{
					UserID: uuid.New(),
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
					},
				}),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "invalid or expired token",
		},
		{
			name:       "wrong signing secret",
			authHeader: "Bearer " + signToken(t, jwt.SigningMethodHS256, "a-different-secret", validClaims(uuid.New())),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "invalid or expired token",
		},
		{
			name:       "unexpected signing method",
			authHeader: "Bearer " + signToken(t, jwt.SigningMethodHS384, cfg.Secret, validClaims(uuid.New())),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "invalid or expired token",
		},
		{
			name: "token missing user id claim",
			authHeader: "Bearer " + signToken(t, jwt.SigningMethodHS256, cfg.Secret,
				&auth.AccessTokenClaims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.Expiry)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
				}),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "token missing user id claim",
		},
		{
			name:           "Success",
			authHeader:     "Bearer " + signToken(t, jwt.SigningMethodHS256, cfg.Secret, validClaims(successUserID)),
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
			wantUserID:     successUserID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// arrange
			var nextCalled bool
			var gotUserID uuid.UUID
			var gotOK bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				gotUserID, gotOK = middleware.UserIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})
			handler := middleware.AuthMiddleware(cfg)(next)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			// act
			handler.ServeHTTP(rec, req)

			// assert
			resp := rec.Result()
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
			assert.Equal(t, tc.wantNextCalled, nextCalled, "next handler call state")
			if tc.wantStatus != http.StatusOK {
				assert.Equal(t, tc.wantBody, readBody(t, resp))
			}
			if tc.wantUserID != uuid.Nil {
				assert.True(t, gotOK, "user id should be present in context")
				assert.Equal(t, tc.wantUserID, gotUserID)
			}
		})
	}
}

// signToken builds and signs a JWT with the given method, secret, and claims.
func signToken(t *testing.T, method jwt.SigningMethod, secret string, claims *auth.AccessTokenClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// readBody reads and trims the response body, matching http.Error's trailing newline.
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return strings.TrimSpace(string(body))
}
