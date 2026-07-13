package main

import (
	"net/http"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	transporthttp "github.com/anthonymartz17/thinkmartz_backend/internal/transport/http"
	"go.uber.org/fx"
)

// NewApp builds the fully wired fx application for ThinkMartz.
func NewApp() *fx.App {
	return fx.New(
		fx.Provide(config.Load),
		fx.Provide(transporthttp.NewHTTPServer),
		fx.Invoke(func(*http.Server) {}),
	)
}
