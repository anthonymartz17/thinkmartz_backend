// Package http provides the HTTP server and (eventually) router for
// the ThinkMartz API.
package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"go.uber.org/fx"
)

// NewHTTPServer builds a bare http.Server and registers its lifecycle
// with fx: OnStart begins listening (non-blocking), OnStop gracefully
// shuts down. No routes are registered yet.
func NewHTTPServer(lc fx.Lifecycle, cfg *config.Config) *http.Server {
	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)

			if err != nil {
				return err
			}
			fmt.Println("ThinkMartz API listening on", srv.Addr)
			go func() {
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
					fmt.Println("server error:", err)
				}
			}()
			return nil

		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("ThinkMartz API shutting down...")
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
