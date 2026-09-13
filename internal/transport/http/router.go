package http

import (
	"net/http"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/fx"
)

// RouterParams groups NewRouter dependencies into a single fx.In struct
// allowing named injection through struct tags disambiguating same-type middlewares.
type RouterParams struct {
	fx.In
	AuthHandler    *auth.Handler
	AuthMiddleware func(http.Handler) http.Handler `name:"authMiddleware"`
}

// NewRouter receives domain handlers as parameters, creates a new chi Router and registers refresh, public and private routes for handlers using group routes
func NewRouter(p RouterParams) chi.Router {
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	r.Group(func(r chi.Router) {
		p.AuthHandler.RegisterRefreshRoutes(r)
	})

	r.Group(func(r chi.Router) {
		p.AuthHandler.RegisterPublicRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(p.AuthMiddleware)
		p.AuthHandler.RegisterProtectedRoutes(r)

	})

	return r
}
