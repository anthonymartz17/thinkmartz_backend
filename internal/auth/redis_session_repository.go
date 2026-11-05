package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// validates RedisSessionRepository implements SessionRepository
var _ SessionRepository = (*RedisSessionRepository)(nil)

// RedisSessionRepository stores refresh-token sessions in Redis, keyed by
// the opaque token itself, with the userID as the value.
type RedisSessionRepository struct {
	client        *redis.Client
	refreshExpiry time.Duration
}

// NewRedisSessionRepository builds and returns a new RedisSessionRepository.
func NewRedisSessionRepository(client *redis.Client, jwtConfig config.JWTConfig) *RedisSessionRepository {
	return &RedisSessionRepository{
		client:        client,
		refreshExpiry: jwtConfig.RefreshExpiry,
	}
}

// Set stores token -> userID, expiring after the configured refresh TTL.
func (r *RedisSessionRepository) Set(ctx context.Context, token string, userID uuid.UUID) error {
	key := refreshTokenKey(token)
	if err := r.client.Set(ctx, key, userID.String(), r.refreshExpiry).Err(); err != nil {
		return fmt.Errorf("set refresh token session: %w", err)
	}
	return nil
}

// GetDel atomically fetches and deletes the session for token, so a refresh
// token can only ever be redeemed once — closing the window where two
// concurrent refresh requests could both succeed on the same old token.
// Returns ErrRefreshTokenNotFound if no session exists.
func (r *RedisSessionRepository) GetDel(ctx context.Context, token string) (uuid.UUID, error) {
	key := refreshTokenKey(token)
	val, err := r.client.GetDel(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		return uuid.Nil, ErrRefreshTokenNotFound
	}

	if err != nil {
		return uuid.Nil, fmt.Errorf("get and delete refresh token session: %w", err)
	}

	userID, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse stored user id: %w", err)
	}

	return userID, nil
}

// Clear removes the session for token. Returns ErrRefreshTokenNotFound if
// no session existed.
func (r *RedisSessionRepository) Clear(ctx context.Context, token string) error {
	key := refreshTokenKey(token)
	deletedCount, err := r.client.Del(ctx, key).Result()

	if err != nil {
		return fmt.Errorf("clear refresh token session: %w", err)
	}

	if deletedCount == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

// refreshTokenKey returns the Redis key under which a refresh token's owning
// user ID is stored.
func refreshTokenKey(token string) string {
	return fmt.Sprintf("session:refresh_token:%s", token)
}
