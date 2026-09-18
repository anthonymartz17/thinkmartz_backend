package post

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// validates RedisRepository implements FeedRepository
var _ FeedRepository = (*RedisRepository)(nil)

// RedisRepository provides a *redis.Client to interact with redis
type RedisRepository struct {
	RedisClient *redis.Client
}

// NewRedisRepo creates a new instance of RedisRepository
func NewRedisRepo(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		RedisClient: client,
	}
}

// AddToFeed adds postIDS to users feed by batching through a RedisClient Pipeliner
func (r *RedisRepository) AddToFeed(ctx context.Context, p Post, followerIDs []uuid.UUID) error {
	pipe := r.RedisClient.Pipeline()

	score := float64(p.CreatedAt.Unix())
	for _, followerID := range followerIDs {
		key := feedKey(followerID)
		pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: p.ID.String()})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("fan out to feeds: %w", err)
	}

	return nil
}

func feedKey(userID uuid.UUID) string {
	return fmt.Sprintf("feed:user:%s", userID.String())
}
