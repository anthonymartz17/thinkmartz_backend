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

func TestGetFollowerCount(t *testing.T) {
	t.Run("zero when no followers", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)

		// act
		count, gotErr := postRepo.GetFollowerCount(ctx, user.ID)

		// assert
		assert.NoError(t, gotErr, "should not fail on zero followers")
		assert.Equal(t, 0, count, "count should be zero when no followers")
	})

	t.Run("context cancelled", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()

		// act
		_, gotErr := postRepo.GetFollowerCount(cancelCtx, user.ID)

		// assert
		assert.Error(t, gotErr, "should fail when context is already cancelled")
	})

	t.Run("increments when a follow is inserted", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		followee := newTestUser(ctx, t, pool)
		follower := newTestUser(ctx, t, pool)

		// act
		_, err := pool.Exec(ctx, `
			INSERT INTO follows (follower_id, followee_id)
			VALUES ($1, $2)
		`, follower.ID, followee.ID)
		require.NoError(t, err, "failed to seed follow")

		// assert
		count, gotErr := postRepo.GetFollowerCount(ctx, followee.ID)
		require.NoError(t, gotErr)
		assert.Equal(t, 1, count, "trigger should increment follower_count on insert")
	})

	t.Run("decrements when a follow is deleted", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		postRepo := post.NewPostgresRepository(pool)
		followee := newTestUser(ctx, t, pool)
		follower := newTestUser(ctx, t, pool)
		_, err := pool.Exec(ctx, `
			INSERT INTO follows (follower_id, followee_id)
			VALUES ($1, $2)
		`, follower.ID, followee.ID)
		require.NoError(t, err, "failed to seed follow")

		count, err := postRepo.GetFollowerCount(ctx, followee.ID)
		require.NoError(t, err)
		require.Equal(t, 1, count, "sanity check: follow should be counted before deleting")

		// act
		_, err = pool.Exec(ctx, `
			DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2
		`, follower.ID, followee.ID)
		require.NoError(t, err, "failed to delete follow")

		// assert
		gotCount, gotErr := postRepo.GetFollowerCount(ctx, followee.ID)
		require.NoError(t, gotErr)
		assert.Equal(t, 0, gotCount, "trigger should decrement follower_count on delete")
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

func TestGetByID(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		ctx := t.Context()

		// act
		gotPost, gotErr := repo.GetByID(ctx, uuid.New())

		// assert
		assert.Nil(t, gotPost)
		assert.ErrorIs(t, gotErr, post.ErrPostNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)
		wantPost := newTestPost(t, user.ID)
		require.NoError(t, repo.Save(ctx, wantPost), "failed to seed post")

		// act
		gotPost, gotErr := repo.GetByID(ctx, wantPost.ID)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, wantPost.ID, gotPost.ID)
		assert.Equal(t, wantPost.UserID, gotPost.UserID)
		assert.Equal(t, wantPost.Content, gotPost.Content)
		assert.Zero(t, gotPost.LikeCount)
		assert.Zero(t, gotPost.CommentCount)
		assert.Equal(t, wantPost.CreatedAt, gotPost.CreatedAt)
		assert.Equal(t, wantPost.UpdatedAt, gotPost.UpdatedAt)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		ctx := t.Context()
		p := &post.Post{ID: uuid.New(), Content: "whatever"}

		// act
		gotErr := repo.Update(ctx, p)

		// assert
		assert.ErrorIs(t, gotErr, post.ErrPostNotFound)
	})

	t.Run("context cancelled", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		p := &post.Post{ID: uuid.New(), Content: "whatever"}

		// act
		gotErr := repo.Update(ctx, p)

		// assert
		assert.Error(t, gotErr, "should fail when context is already cancelled")
	})

	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)
		p := newTestPost(t, user.ID)
		require.NoError(t, repo.Save(ctx, p), "failed to seed post")
		wantContent := "updated content"
		updated := &post.Post{ID: p.ID, Content: wantContent}

		// act
		gotErr := repo.Update(ctx, updated)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, wantContent, updated.Content)
		assert.Equal(t, user.ID, updated.UserID)
		assert.True(t, updated.CreatedAt.Equal(p.CreatedAt), "CreatedAt should not change")
		assert.True(t, updated.UpdatedAt.After(p.UpdatedAt), "UpdatedAt should move forward")
		assert.Zero(t, updated.LikeCount)
		assert.Zero(t, updated.CommentCount)
	})
}

func TestSaveLike(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)

		// act
		gotErr := repo.SaveLike(ctx, post.Like{UserID: user.ID, PostID: uuid.New()})

		// assert
		assert.ErrorIs(t, gotErr, post.ErrPostNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)
		p := newTestPost(t, user.ID)
		require.NoError(t, repo.Save(ctx, p), "failed to seed post")

		// act
		gotErr := repo.SaveLike(ctx, post.Like{UserID: user.ID, PostID: p.ID})

		// assert
		require.NoError(t, gotErr)
		gotPost, err := repo.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, gotPost.LikeCount, "like_count should be incremented by the trigger")
	})

	t.Run("duplicate like is a no-op", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)
		p := newTestPost(t, user.ID)
		require.NoError(t, repo.Save(ctx, p), "failed to seed post")
		like := post.Like{UserID: user.ID, PostID: p.ID}
		require.NoError(t, repo.SaveLike(ctx, like), "first like should succeed")

		// act
		gotErr := repo.SaveLike(ctx, like)

		// assert
		require.NoError(t, gotErr, "liking the same post twice should not error")
		gotPost, err := repo.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, gotPost.LikeCount, "duplicate like should not double-count")
	})
}

func TestDeleteLike(t *testing.T) {
	t.Run("no-op when like does not exist", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)

		// act
		gotErr := repo.DeleteLike(ctx, post.Like{UserID: user.ID, PostID: uuid.New()})

		// assert
		assert.NoError(t, gotErr)
	})

	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)
		p := newTestPost(t, user.ID)
		require.NoError(t, repo.Save(ctx, p), "failed to seed post")
		like := post.Like{UserID: user.ID, PostID: p.ID}
		require.NoError(t, repo.SaveLike(ctx, like), "failed to seed like")

		likedPost, err := repo.GetByID(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, 1, likedPost.LikeCount, "sanity check: like should be counted before deleting")

		// act
		gotErr := repo.DeleteLike(ctx, like)

		// assert
		require.NoError(t, gotErr)
		gotPost, err := repo.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, gotPost.LikeCount, "like_count should be decremented by the trigger")
	})

	t.Run("no-op when like already deleted", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := post.NewPostgresRepository(pool)
		user := newTestUser(ctx, t, pool)
		p := newTestPost(t, user.ID)
		require.NoError(t, repo.Save(ctx, p), "failed to seed post")
		like := post.Like{UserID: user.ID, PostID: p.ID}
		require.NoError(t, repo.SaveLike(ctx, like), "failed to seed like")
		require.NoError(t, repo.DeleteLike(ctx, like), "first delete should succeed")

		// act
		gotErr := repo.DeleteLike(ctx, like)

		// assert
		require.NoError(t, gotErr, "deleting an already-removed like should not error")
		gotPost, err := repo.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, gotPost.LikeCount, "like_count should not go negative")
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
