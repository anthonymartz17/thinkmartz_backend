// Package redis handles redis client
package redis

import (
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient builds and returns a new *redis.Client
func NewRedisClient(cfg config.RedisConfig) *redis.Client {

	return redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
	})
}
