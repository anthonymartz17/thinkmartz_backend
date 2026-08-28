// Package logging handles creating the application's structured logger.
package logging

import (
	"context"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewLogger constructs the application's structured logger.
func NewLogger(lc fx.Lifecycle, cfg config.AppConfig) (*zap.Logger, error) {

	var (
		err    error
		logger *zap.Logger
	)

	if cfg.Env == "production" {
		logger, err = zap.NewProduction()

		if err != nil {
			return nil, err
		}

	} else {

		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return logger.Sync()
		},
	})

	return logger, nil
}
