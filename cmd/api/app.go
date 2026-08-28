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
		fx.Provide(config.NewAppConfig),
		fx.Provide(config.NewDBConfig),
		fx.Provide(config.NewRedisConfig),
		fx.Provide(config.NewJWTConfig),
		fx.Provide(logging.NewLogger),
		fx.Provide(transporthttp.NewHTTPServer),
		fx.Provide(database.NewPostgresPool),
		fx.Provide(redis.NewRedisClient),
		fx.Provide(fx.Annotate(auth.NewRepository, fx.As(new(auth.UserRepository)))),
		fx.Provide(fx.Annotate(auth.NewTokenService, fx.As(new(auth.TokenIssuer)))),
		fx.Provide(fx.Annotate(auth.NewService, fx.As(new(auth.Authenticator)))),
		fx.Provide(auth.NewHandler),
		fx.Provide(transporthttp.NewRouter),
		fx.Invoke(func(*http.Server) {}),
		fx.Invoke(func(*pgxpool.Pool) {}),
	)

}
