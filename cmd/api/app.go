package main

import (
	"net/http"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	transporthttp "github.com/anthonymartz17/thinkmartz_backend/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// NewApp builds the fully wired fx application for ThinkMartz.
func NewApp() *fx.App {
	return fx.New(
		fx.Provide(config.Load),
		fx.Provide(transporthttp.NewHTTPServer),
		fx.Provide(database.NewPostgresPool),
		fx.Invoke(func(*http.Server) {}),
		fx.Invoke(func(*pgxpool.Pool) {}),
	)
}
