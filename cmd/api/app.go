package main

import (
	"net/http"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	"github.com/anthonymartz17/thinkmartz_backend/internal/logging"
	"github.com/anthonymartz17/thinkmartz_backend/internal/redis"
	transporthttp "github.com/anthonymartz17/thinkmartz_backend/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// NewApp builds the fully wired fx application for ThinkMartz.
func NewApp() *fx.App {
	return fx.New(
		fx.Provide(config.Load),
		fx.Provide(logging.NewLogger),
		fx.Provide(transporthttp.NewHTTPServer),
		fx.Provide(database.NewPostgresPool),
		fx.Provide(redis.NewRedisClient),
		fx.Provide(auth.NewService),
		fx.Provide(auth.NewTokenService),
		fx.Provide(auth.NewRepository),
		fx.Provide(auth.NewHandler),
		fx.Invoke(func(*http.Server) {}),
		fx.Invoke(func(*pgxpool.Pool) {}),
	)

}
