// Package feed fans posts out to followers' feeds in Redis.
package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	_feedPageSize = 20
	// _feedMaxSize keeps two pages of IDs so page one stays full even if
	// hydration deletes a few posts.
	_feedMaxSize = 2 * _feedPageSize
	_feedTTL     = 7 * 24 * time.Hour
)

// _addToFeedScript adds a post to a feed only if the feed key already exists,
// then trims the feed to the newest ARGV[3] posts, all in one atomic step.
// A missing key is skipped so fan-out never creates a partial feed; the read
// path rebuilds missing feeds from Postgres.
//
// KEYS[1]: feed key
// ARGV[1]: score (post created_at in Unix microseconds)
// ARGV[2]: post ID
// ARGV[3]: max feed size
//
// Returns 1 if the feed was updated, 0 if the key was missing.
var _addToFeedScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 1 then
    redis.call("ZADD", KEYS[1], ARGV[1], ARGV[2])
    redis.call("ZREMRANGEBYRANK", KEYS[1], 0, -(tonumber(ARGV[3]) + 1))
    return 1
end

return 0
`)

// validates RedisRepository implements post.FeedWriter interface
var _ post.FeedWriter = (*RedisRepository)(nil)

// validates RedisRepository implements Cache interface
var _ Cache = (*RedisRepository)(nil)

// RedisRepository provides a *redis.Client to interact with redis
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepo creates a new instance of RedisRepository
func NewRedisRepo(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

// AddToFeed adds a post to each follower's feed that already exists, trimming
// it to the _feedMaxSize newest posts. Missing feeds are skipped, not created,
// so fan-out never leaves a partial feed; the read path rebuilds feed on demand.
func (r *RedisRepository) AddToFeed(ctx context.Context, p post.Post, followerIDs []uuid.UUID) error {

	if len(followerIDs) == 0 {
		return nil
	}

	score := float64(p.CreatedAt.UnixMicro())

	_, err := r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, followerID := range followerIDs {
			key := Key(followerID)
			_addToFeedScript.Eval(ctx, pipe, []string{key}, score, p.ID.String(), _feedMaxSize)

		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("fan out to feeds: %w", err)
	}

	return nil
}

// RemoveFromFeeds removes a post from each follower's feed.
// A missing member is not an error (ZREM returns 0).
func (r *RedisRepository) RemoveFromFeeds(ctx context.Context, postID uuid.UUID, followerIDs []uuid.UUID) error {
	if len(followerIDs) == 0 {
		return nil
	}

	member := postID.String()
	_, err := r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, followerID := range followerIDs {
			pipe.ZRem(ctx, Key(followerID), member)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("remove from feeds: %w", err)
	}

	return nil
}

// Get returns the newest _feedPageSize post IDs in the user's feed and
// refreshes its TTL. A missing feed returns an empty slice and no error.
func (r *RedisRepository) Get(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	key := Key(userID)

	var rangeCmd *redis.StringSliceCmd
	_, err := r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		rangeCmd = pipe.ZRevRange(ctx, key, 0, _feedPageSize-1)
		pipe.Expire(ctx, key, _feedTTL)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("read feed: %w", err)
	}

	members := rangeCmd.Val()
	ids := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		id, err := uuid.Parse(m)
		if err != nil {
			return nil, fmt.Errorf("parse feed member %q: %w", m, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// Set writes the given entries to the user's feed, scored by created_at,
// and sets its TTL. Used to rebuild a feed on a cache miss.
func (r *RedisRepository) Set(ctx context.Context, userID uuid.UUID, feed []Entry) error {
	if len(feed) == 0 {
		return nil
	}
	key := Key(userID)

	members := make([]redis.Z, 0, len(feed))
	for _, entry := range feed {
		redisZ := redis.Z{
			Score:  float64(entry.CreatedAt.UnixMicro()),
			Member: entry.PostID.String(),
		}

		members = append(members, redisZ)
	}

	_, err := r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZAdd(ctx, key, members...)
		pipe.Expire(ctx, Key(userID), _feedTTL)
		return nil
	})

	if err != nil {
		return fmt.Errorf("set feed entries: %w", err)
	}
	return nil
}

// Key returns the Redis key holding a user's feed sorted set.
func Key(userID uuid.UUID) string {
	return fmt.Sprintf("feed:user:%s", userID.String())
}
