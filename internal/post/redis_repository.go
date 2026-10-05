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

const _feedMaxSize = 50

// AddToFeed adds postIDS to users feed by batching through a RedisClient Pipeliner,
// trimming each feed to the _feedMaxSize most recent posts so it doesn't grow unbounded.
func (r *RedisRepository) AddToFeed(ctx context.Context, p Post, followerIDs []uuid.UUID) error {
	score := float64(p.CreatedAt.Unix())

	_, err := r.RedisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, followerID := range followerIDs {
			key := FeedKey(followerID)
			pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: p.ID.String()})
			pipe.ZRemRangeByRank(ctx, key, 0, -(_feedMaxSize + 1))
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("fan out to feeds: %w", err)
	}

	return nil
}

// FeedKey returns the Redis key holding a user's feed sorted set.
func FeedKey(userID uuid.UUID) string {
	return fmt.Sprintf("feed:user:%s", userID.String())
}
