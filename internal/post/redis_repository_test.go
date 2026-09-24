package post_test

import (
	"context"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	redisClient "github.com/anthonymartz17/thinkmartz_backend/internal/redis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToFeed(t *testing.T) {
	t.Run("adds post to follower's feed", func(t *testing.T) {
		// arrange
		redisRepo := newTestRedisRepo(t)
		ctx := t.Context()

		followerID := uuid.New()
		p := post.Post{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
		}

		// act
		gotErr := redisRepo.AddToFeed(ctx, p, []uuid.UUID{followerID})

		// assert
		require.NoError(t, gotErr)

		key := post.FeedKey(followerID)
		score, err := redisRepo.RedisClient.ZScore(ctx, key, p.ID.String()).Result()
		require.NoError(t, err, "post should exist in follower's feed")
		assert.Equal(t, float64(p.CreatedAt.Unix()), score, "score should match post's CreatedAt")
	})

	t.Run("context cancelled", func(t *testing.T) {
		// arrange
		redisRepo := newTestRedisRepo(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		followerID := uuid.New()
		p := post.Post{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
		}

		// act
		gotErr := redisRepo.AddToFeed(ctx, p, []uuid.UUID{followerID})

		// assert
		assert.Error(t, gotErr)
	})
}

func newTestRedisRepo(t *testing.T) *post.RedisRepository {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	return post.NewRedisRepo(redisClient.NewRedisClient(cfg.Redis))
}
