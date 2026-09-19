package main

import (
	"net/http"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	"github.com/anthonymartz17/thinkmartz_backend/internal/logging"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/anthonymartz17/thinkmartz_backend/internal/redis"
	httpTransport "github.com/anthonymartz17/thinkmartz_backend/internal/transport/http"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// NewApp builds the fully wired fx application for ThinkMartz.
func NewApp() *fx.App {
	return fx.New(
		config.Module,
		auth.Module,
		middleware.Module,
		httpTransport.Module,
		post.Module,
		fx.Provide(logging.NewLogger),
		fx.Provide(database.NewPostgresPool),
		fx.Provide(redis.NewRedisClient),
		fx.Invoke(func(*http.Server) {}),
		fx.Invoke(func(*pgxpool.Pool) {}),
	)

}
