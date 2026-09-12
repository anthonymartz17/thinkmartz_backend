package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config" // adjust import path
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
)

func TestAuthMiddleware_Scratch(_ *testing.T) {
	cfg := config.JWTConfig{
		Secret:        "fake-secret",
		Expiry:        15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	// this is "next" — what runs if AuthMiddleware lets the request through
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.AuthMiddleware(cfg)(next)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer some-fake-token-for-now")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	fmt.Println("status code:", rec.Code)
	fmt.Println("body:", rec.Body.String())
}
