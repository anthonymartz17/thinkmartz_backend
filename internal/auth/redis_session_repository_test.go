package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	redisClient "github.com/anthonymartz17/thinkmartz_backend/internal/redis"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		repo, rdb := newTestSessionRepo(t)
		userID := uuid.New()
		token := "some-opaque-token"

		// act
		gotErr := repo.Set(ctx, token, userID)

		// assert
		require.NoError(t, gotErr, "should not fail on success case")
		key := fmt.Sprintf("session:refresh_token:%s", token)
		storedUserID, err := rdb.Get(ctx, key).Result()
		assert.NoError(t, err, "should not fail on success case")
		assert.Equal(t, userID.String(), storedUserID, "stored value should be the given user's ID")
	})

	t.Run("Fails when redis is unreachable", func(t *testing.T) {
		// arrange
		repo, rdb := newTestSessionRepo(t)
		require.NoError(t, rdb.Close(), "should close redis client cleanly for setup")

		// act
		gotErr := repo.Set(t.Context(), "some-opaque-token", uuid.New())

		// assert
		assert.Error(t, gotErr, "should return an error when redis is unreachable")
	})
}

func TestGetDel(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		repo, rdb := newTestSessionRepo(t)
		userID := uuid.New()
		token := "old-opaque-token"
		require.NoError(t, repo.Set(ctx, token, userID), "should not fail to set up a session")

		// act
		gotUserID, gotErr := repo.GetDel(ctx, token)

		// assert
		assert.NoError(t, gotErr, "should not fail on success case")
		assert.Equal(t, userID, gotUserID, "should return the session's user ID")

		key := fmt.Sprintf("session:refresh_token:%s", token)
		_, err := rdb.Get(ctx, key).Result()
		assert.ErrorIs(t, err, redis.Nil, "token should be deleted atomically, so it can't be redeemed twice")
	})

	t.Run("Fails when token is not found", func(t *testing.T) {
		// arrange
		repo, _ := newTestSessionRepo(t)

		// act
		gotUserID, gotErr := repo.GetDel(t.Context(), "non-existent-opaque-token")

		// assert
		assert.Equal(t, uuid.Nil, gotUserID, "should return uuid.Nil when token is not found")
		assert.ErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound)
	})

	t.Run("Fails when stored value is malformed", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		repo, rdb := newTestSessionRepo(t)
		token := "malformed-value-opaque-token"
		key := fmt.Sprintf("session:refresh_token:%s", token)

		require.NoError(t, rdb.Set(ctx, key, "not-a-valid-uuid", time.Minute).Err(),
			"should set up a malformed stored value")

		// act
		gotUserID, gotErr := repo.GetDel(ctx, token)

		// assert
		assert.Equal(t, uuid.Nil, gotUserID)
		assert.Error(t, gotErr, "should return an error when the stored value is not a valid UUID")
		assert.NotErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound, "malformed data should not be mistaken for not-found")
	})

	t.Run("Fails when redis is unreachable", func(t *testing.T) {
		// arrange
		repo, rdb := newTestSessionRepo(t)
		require.NoError(t, rdb.Close(), "should close redis client cleanly for setup")

		// act
		gotUserID, gotErr := repo.GetDel(t.Context(), "any-opaque-token")

		// assert
		assert.Equal(t, uuid.Nil, gotUserID)
		assert.Error(t, gotErr, "should return an error when redis is unreachable")
		assert.NotErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound, "connection failure should not be reported as not-found")
	})
}

func TestClear(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		repo, rdb := newTestSessionRepo(t)
		token := "old-opaque-token"
		require.NoError(t, repo.Set(ctx, token, uuid.New()), "should not fail to set up a session")

		// act
		gotErr := repo.Clear(ctx, token)

		// assert
		assert.NoError(t, gotErr, "should not fail on success case")

		key := fmt.Sprintf("session:refresh_token:%s", token)
		_, err := rdb.Get(ctx, key).Result()
		assert.ErrorIs(t, err, redis.Nil, "token should no longer exist in redis after clearing")
	})

	t.Run("Fails when token is not found", func(t *testing.T) {
		// arrange
		repo, _ := newTestSessionRepo(t)

		// act
		gotErr := repo.Clear(t.Context(), "non-existent-opaque-token")

		// assert
		assert.ErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound)
	})

	t.Run("Fails when redis is unreachable", func(t *testing.T) {
		// arrange
		repo, rdb := newTestSessionRepo(t)
		require.NoError(t, rdb.Close(), "should close redis client cleanly for setup")

		// act
		gotErr := repo.Clear(t.Context(), "any-opaque-token")

		// assert
		assert.Error(t, gotErr, "should return an error when redis is unreachable")
		assert.NotErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound, "connection failure should not be reported as not-found")
	})
}

func newTestSessionRepo(t *testing.T) (*auth.RedisSessionRepository, *redis.Client) {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "config should load on setup")

	client := redisClient.NewRedisClient(cfg.Redis)
	repo := auth.NewRedisSessionRepository(client, cfg.JWT)

	return repo, client
}
