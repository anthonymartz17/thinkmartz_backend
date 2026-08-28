package http

import (
	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(authHandler *auth.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Group(func(r chi.Router) {
		//r.Use(RefreshMiddleware) // once built — validates the refresh cookie, not a standard access token
		authHandler.RegisterRefreshRoutes(r)
	})

	r.Group(func(r chi.Router) {
		authHandler.RegisterPublicRoutes(r)
	})

	r.Group(func(r chi.Router) {
		// r.Use(AuthMiddleware)
		authHandler.RegisterProtectedRoutes(r)

	})

	return r
}
