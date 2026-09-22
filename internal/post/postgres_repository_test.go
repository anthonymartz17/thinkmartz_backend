package post_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)
		p := newTestPost(t, user.ID)

		// act
		gotErr := repo.Save(ctx, p)

		// assert
		assert.NoError(t, gotErr)
		assert.NotEqual(t, uuid.Nil, p.ID, "ID should be populated after save")
		assert.False(t, p.CreatedAt.IsZero(), "CreatedAt should be populated after save")
		assert.False(t, p.UpdatedAt.IsZero(), "UpdatedAt should be populated after save")
	})

	t.Run("Fails when user does not exist", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		p := newTestPost(t, uuid.New())

		// act
		gotErr := repo.Save(ctx, p)

		// assert
		assert.Error(t, gotErr, "should fail when the referenced user does not exist")
	})
}

func TestCountFollowers(t *testing.T) {
	t.Run("no followers", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		userID := uuid.New()
		ctx := t.Context()

		// act
		count, gotErr := postRepo.CountFollowers(ctx, userID)

		// assert
		assert.NoError(t, gotErr, "should not fail on zero followers")
		assert.Equal(t, 0, count, "count should be zero when no followers")

	})

	t.Run("context cancelled", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		userID := uuid.New()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		// act
		_, gotErr := postRepo.CountFollowers(ctx, userID)

		// assert
		assert.Error(t, gotErr, "should fail when context is already cancelled")

	})

	t.Run("multiple followers", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		ctx := t.Context()

		followee := newTestUser(ctx, t, pool)

		const followerCount = 5
		for range followerCount {
			follower := newTestUser(ctx, t, pool)
			_, err := pool.Exec(ctx, `
			INSERT INTO follows (follower_id, followee_id)
			VALUES ($1, $2)
		`, follower.ID, followee.ID)
			require.NoError(t, err, "failed to seed follow")
		}

		// act
		count, gotErr := postRepo.CountFollowers(ctx, followee.ID)

		// assert
		assert.NoError(t, gotErr, "should not fail on multiple followers")
		assert.Equal(t, followerCount, count, "count should match number of seeded followers")
	})
}

func TestGetFollowersByID(t *testing.T) {
	t.Run("no followers", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		userID := uuid.New()
		ctx := t.Context()

		// act
		followers, gotErr := postRepo.GetFollowersByID(ctx, userID)

		// assert
		assert.NoError(t, gotErr, "should not fail on zero followers")
		assert.Len(t, followers, 0, "count should be zero when no followers")

	})

	t.Run("context cancelled", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		userID := uuid.New()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		// act
		_, gotErr := postRepo.GetFollowersByID(ctx, userID)

		// assert
		assert.Error(t, gotErr, "should fail when context is already cancelled")

	})

	t.Run("returns followers", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		ctx := t.Context()

		var userIDs []uuid.UUID
		for i := 0; i < 3; i++ {
			user := newTestUser(ctx, t, pool)
			userIDs = append(userIDs, user.ID)
		}
		followeeID := userIDs[2]
		wantFollowerIDs := userIDs[:2] // first two are followers

		for _, followerID := range wantFollowerIDs {
			_, err := pool.Exec(ctx, `
			INSERT INTO follows (follower_id, followee_id)
			VALUES ($1, $2)
		`, followerID, followeeID)
			require.NoError(t, err, "failed to seed follow")
		}

		// act
		gotFollowerIDs, gotErr := postRepo.GetFollowersByID(ctx, followeeID)

		// assert
		assert.NoError(t, gotErr)
		assert.ElementsMatch(t, wantFollowerIDs, gotFollowerIDs, "should return exactly the seeded followers")
	})
}

func newTestPost(t *testing.T, userID uuid.UUID) *post.Post {
	t.Helper()

	return &post.Post{
		UserID:  userID,
		Content: "some random content",
	}
}

func newTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *auth.User {
	t.Helper()

	authRepo := auth.NewPostgresRepository(pool)

	user := &auth.User{
		Email:        fmt.Sprintf("test-%s@email.com", uuid.New().String()),
		Username:     fmt.Sprintf("%s-%s", t.Name(), uuid.New().String()[:8]), // [:8] keeps username short since uuid.New().String() produces a 36-character string
		PasswordHash: "fake-hashed-password",
	}
	require.NoError(t, authRepo.Save(ctx, user), "failed to seed test user")

	t.Cleanup(func() {
		// ON DELETE CASCADE on posts.user_id also removes any post saved against
		// this user, so no separate post cleanup is needed.
		_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
		if err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	})

	return user
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	pool, err := database.NewPool(t.Context(), cfg.DB)
	require.NoError(t, err, "failed to create database pool")

	return pool
}
