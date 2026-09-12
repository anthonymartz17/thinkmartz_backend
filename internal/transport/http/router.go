package http

import (
	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter receives domain handlers as parameters, creates a new chi Router and registers refresh, public and private routes for handlers using group routes
func NewRouter(authHandler *auth.Handler, jwtConfig config.JWTConfig) chi.Router {
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	r.Group(func(r chi.Router) {
		//r.Use(RefreshMiddleware) // once built — validates the refresh cookie, not a standard access token
		authHandler.RegisterRefreshRoutes(r)
	})

	r.Group(func(r chi.Router) {
		authHandler.RegisterPublicRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtConfig))
		authHandler.RegisterProtectedRoutes(r)

	})

	return r
}
