// Package database provides the PostgreSQL connection pool for the
// ThinkMartz API.
package database

import (
	"context"
	"fmt"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// NewPostgresPool builds a pgx connection pool and registers its
// lifecycle with fx: OnStart pings the database to fail fast if it's
// unreachable, OnStop closes the pool cleanly.
func NewPostgresPool(lc fx.Lifecycle, cfg *config.Config) (*pgxpool.Pool, error) {

	poolConfig, err := pgxpool.ParseConfig(connectionString(&cfg.DB))

	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	poolConfig.MaxConns = cfg.DB.MaxConns
	poolConfig.MinConns = cfg.DB.MinConns

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {

			if err := pool.Ping(ctx); err != nil {
				return fmt.Errorf("ping postgres: %w", err)
			}
			fmt.Println("Connected to PostgreSQL")
			return nil

		},
		OnStop: func(_ context.Context) error {
			pool.Close()
			fmt.Println("PostgreSQL connection closed")
			return nil
		},
	})

	return pool, nil
}

func connectionString(cfg *config.DBConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)
}
