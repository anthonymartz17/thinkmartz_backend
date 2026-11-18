package user_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	"github.com/anthonymartz17/thinkmartz_backend/internal/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveFollow(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := user.NewPostgresRepository(pool)
		follower := newTestUser(ctx, t, pool)
		followee := newTestUser(ctx, t, pool)

		// act
		gotErr := repo.SaveFollow(ctx, user.Follow{FollowerID: follower.ID, FolloweeID: followee.ID})

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, 1, countFollows(ctx, t, pool, follower.ID, followee.ID))
	})

	t.Run("saving an existing follow is a no-op", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := user.NewPostgresRepository(pool)
		follower := newTestUser(ctx, t, pool)
		followee := newTestUser(ctx, t, pool)
		f := user.Follow{FollowerID: follower.ID, FolloweeID: followee.ID}
		require.NoError(t, repo.SaveFollow(ctx, f), "failed to set up the initial follow")

		// act
		gotErr := repo.SaveFollow(ctx, f)

		// assert
		assert.NoError(t, gotErr, "re-saving an existing follow should succeed without duplicating it")
		assert.Equal(t, 1, countFollows(ctx, t, pool, follower.ID, followee.ID))
	})

	t.Run("fails when followee does not exist", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := user.NewPostgresRepository(pool)
		follower := newTestUser(ctx, t, pool)

		// act
		gotErr := repo.SaveFollow(ctx, user.Follow{FollowerID: follower.ID, FolloweeID: uuid.New()})

		// assert
		assert.ErrorIs(t, gotErr, user.ErrUserNotFound)
	})

	t.Run("fails when follower does not exist", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := user.NewPostgresRepository(pool)
		followee := newTestUser(ctx, t, pool)

		// act
		gotErr := repo.SaveFollow(ctx, user.Follow{FollowerID: uuid.New(), FolloweeID: followee.ID})

		// assert
		// Only the followee_id foreign key is translated to ErrUserNotFound
		// today, so a missing follower surfaces as a generic wrapped error.
		assert.Error(t, gotErr)
		assert.NotErrorIs(t, gotErr, user.ErrUserNotFound)
	})

	t.Run("context cancelled", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := user.NewPostgresRepository(pool)
		follower := newTestUser(ctx, t, pool)
		followee := newTestUser(ctx, t, pool)
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()

		// act
		gotErr := repo.SaveFollow(cancelCtx, user.Follow{FollowerID: follower.ID, FolloweeID: followee.ID})

		// assert
		assert.Error(t, gotErr, "should fail when context is already cancelled")
	})
}

func countFollows(ctx context.Context, t *testing.T, pool *pgxpool.Pool, followerID, followeeID uuid.UUID) int {
	t.Helper()

	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM follows WHERE follower_id = $1 AND followee_id = $2`,
		followerID, followeeID,
	).Scan(&count)
	require.NoError(t, err, "failed to count follows row")

	return count
}

func newTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *auth.User {
	t.Helper()

	authRepo := auth.NewPostgresRepository(pool)

	u := &auth.User{
		Email:        fmt.Sprintf("test-%s@email.com", uuid.New().String()),
		Username:     fmt.Sprintf("%s-%s", t.Name(), uuid.New().String()[:8]),
		PasswordHash: "fake-hashed-password",
	}
	require.NoError(t, authRepo.Save(ctx, u), "failed to seed test user")

	t.Cleanup(func() {
		// ON DELETE CASCADE on follows' follower_id/followee_id also removes
		// any follow rows referencing this user, so no separate cleanup
		// is needed for those.
		_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.ID)
		if err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	})

	return u
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	pool, err := database.NewPool(t.Context(), cfg.DB)
	require.NoError(t, err, "failed to create database pool")

	return pool
}
