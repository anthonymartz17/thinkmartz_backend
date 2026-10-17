package feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/feed"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	redisClient "github.com/anthonymartz17/thinkmartz_backend/internal/redis"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToFeed(t *testing.T) {
	t.Run("adds post to follower's feed", func(t *testing.T) {
		// arrange
		redisRepo, client := newTestRedisRepo(t)
		ctx := t.Context()

		followerID := uuid.New()
		p := post.Post{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
		}
		key := feed.Key(followerID)
		require.NoError(t, client.ZAdd(ctx, key, redis.Z{Score: 0, Member: uuid.NewString()}).Err(),
			"seed the feed key so the fan-out script doesn't skip it")

		// act
		gotErr := redisRepo.AddToFeed(ctx, p, []uuid.UUID{followerID})

		// assert
		require.NoError(t, gotErr)
		score, err := client.ZScore(ctx, key, p.ID.String()).Result()
		require.NoError(t, err, "post should exist in follower's feed")
		assert.Equal(t, float64(p.CreatedAt.UnixMicro()), score, "score should match post's CreatedAt")
	})

	t.Run("context cancelled", func(t *testing.T) {
		// arrange
		redisRepo, _ := newTestRedisRepo(t)
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

func TestGet(t *testing.T) {
	t.Run("returns newest first", func(t *testing.T) {
		// arrange
		redisRepo, client := newTestRedisRepo(t)
		ctx := t.Context()
		userID := uuid.New()
		key := feed.Key(userID)

		oldest := uuid.New()
		middle := uuid.New()
		newest := uuid.New()
		now := time.Now()
		seedMember(ctx, t, client, key, oldest, now.Add(-2*time.Hour))
		seedMember(ctx, t, client, key, middle, now.Add(-1*time.Hour))
		seedMember(ctx, t, client, key, newest, now)

		// act
		gotIDs, gotErr := redisRepo.Get(ctx, userID)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, []uuid.UUID{newest, middle, oldest}, gotIDs, "should be ordered newest to oldest")
	})

	t.Run("returns at most a page", func(t *testing.T) {
		// arrange
		redisRepo, client := newTestRedisRepo(t)
		ctx := t.Context()
		userID := uuid.New()
		key := feed.Key(userID)

		const seeded = 25
		now := time.Now()
		ids := make([]uuid.UUID, seeded)
		for i := range seeded {
			id := uuid.New()
			ids[i] = id
			// earlier index = older: i=0 is oldest, i=24 is newest
			seedMember(ctx, t, client, key, id, now.Add(time.Duration(i)*time.Second))
		}

		// act
		gotIDs, gotErr := redisRepo.Get(ctx, userID)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotIDs, 20, "should cap at one page")

		wantNewest20 := make([]uuid.UUID, 20)
		for i := range wantNewest20 {
			wantNewest20[i] = ids[seeded-1-i] // newest (highest index) first
		}
		assert.Equal(t, wantNewest20, gotIDs, "should return the 20 newest entries")
	})

	t.Run("missing key", func(t *testing.T) {
		// arrange
		redisRepo, _ := newTestRedisRepo(t)
		ctx := t.Context()

		// act
		gotIDs, gotErr := redisRepo.Get(ctx, uuid.New())

		// assert
		assert.NoError(t, gotErr)
		assert.Empty(t, gotIDs)
	})

	t.Run("refreshes the TTL", func(t *testing.T) {
		// arrange
		redisRepo, client := newTestRedisRepo(t)
		ctx := t.Context()
		userID := uuid.New()
		key := feed.Key(userID)
		seedMember(ctx, t, client, key, uuid.New(), time.Now())
		require.NoError(t, client.Expire(ctx, key, 5*time.Second).Err(), "start with a short TTL")

		// act
		_, gotErr := redisRepo.Get(ctx, userID)

		// assert
		require.NoError(t, gotErr)
		ttl, err := client.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.InDelta(t, (7 * 24 * time.Hour).Seconds(), ttl.Seconds(), 5, "TTL should be refreshed to ~7 days")
	})

	t.Run("invalid member", func(t *testing.T) {
		// arrange
		redisRepo, client := newTestRedisRepo(t)
		ctx := t.Context()
		userID := uuid.New()
		key := feed.Key(userID)
		require.NoError(t, client.ZAdd(ctx, key, redis.Z{Score: 1, Member: "not-a-uuid"}).Err())

		// act
		_, gotErr := redisRepo.Get(ctx, userID)

		// assert
		assert.Error(t, gotErr)
	})
}

// seedMember adds a single raw sorted-set entry, bypassing Set entirely, so
// Get's tests never depend on Set's correctness.
func seedMember(ctx context.Context, t *testing.T, client *redis.Client, key string, id uuid.UUID, createdAt time.Time) {
	t.Helper()
	err := client.ZAdd(ctx, key, redis.Z{Score: float64(createdAt.UnixMicro()), Member: id.String()}).Err()
	require.NoError(t, err, "failed to seed feed member")
}

func TestSet(t *testing.T) {
	t.Run("writes entries with their scores", func(t *testing.T) {
		// arrange
		redisRepo, client := newTestRedisRepo(t)
		ctx := t.Context()
		userID := uuid.New()
		key := feed.Key(userID)

		entryA := feed.Entry{PostID: uuid.New(), CreatedAt: time.Now().Add(-time.Hour)}
		entryB := feed.Entry{PostID: uuid.New(), CreatedAt: time.Now()}

		// act
		gotErr := redisRepo.Set(ctx, userID, []feed.Entry{entryA, entryB})

		// assert
		require.NoError(t, gotErr)

		scoreA, err := client.ZScore(ctx, key, entryA.PostID.String()).Result()
		require.NoError(t, err)
		assert.Equal(t, float64(entryA.CreatedAt.UnixMicro()), scoreA)

		scoreB, err := client.ZScore(ctx, key, entryB.PostID.String()).Result()
		require.NoError(t, err)
		assert.Equal(t, float64(entryB.CreatedAt.UnixMicro()), scoreB)
	})

	t.Run("sets the TTL", func(t *testing.T) {
		// arrange
		redisRepo, client := newTestRedisRepo(t)
		ctx := t.Context()
		userID := uuid.New()
		key := feed.Key(userID)
		entry := feed.Entry{PostID: uuid.New(), CreatedAt: time.Now()}

		// act
		gotErr := redisRepo.Set(ctx, userID, []feed.Entry{entry})

		// assert
		require.NoError(t, gotErr)
		ttl, err := client.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.InDelta(t, (7 * 24 * time.Hour).Seconds(), ttl.Seconds(), 5, "TTL should be set to ~7 days")
	})

	t.Run("empty feed is a no-op", func(t *testing.T) {
		// arrange
		redisRepo, client := newTestRedisRepo(t)
		ctx := t.Context()
		userID := uuid.New()
		key := feed.Key(userID)

		// act
		gotErr := redisRepo.Set(ctx, userID, nil)

		// assert
		require.NoError(t, gotErr)
		exists, err := client.Exists(ctx, key).Result()
		require.NoError(t, err)
		assert.Zero(t, exists, "key should never be created for an empty feed")
	})

	t.Run("context cancelled", func(t *testing.T) {
		// arrange
		redisRepo, _ := newTestRedisRepo(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		entry := feed.Entry{PostID: uuid.New(), CreatedAt: time.Now()}

		// act
		gotErr := redisRepo.Set(ctx, uuid.New(), []feed.Entry{entry})

		// assert
		assert.Error(t, gotErr)
	})
}

func newTestRedisRepo(t *testing.T) (*feed.RedisRepository, *redis.Client) {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	client := redisClient.NewRedisClient(cfg.Redis)
	return feed.NewRedisRepo(client), client
}
