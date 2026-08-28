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
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

// NewHTTPServer builds a bare http.Server and registers its lifecycle
// with fx: OnStart begins listening (non-blocking), OnStop gracefully
// shuts down. 
func NewHTTPServer(lc fx.Lifecycle, cfg config.AppConfig, r chi.Router) *http.Server {

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
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
