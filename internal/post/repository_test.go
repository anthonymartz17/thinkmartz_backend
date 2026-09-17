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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		repo := newTestRepository(t)
		user := newTestUser(ctx, t)
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
		repo := newTestRepository(t)
		p := newTestPost(t, uuid.New())

		// act
		gotErr := repo.Save(ctx, p)

		// assert
		assert.Error(t, gotErr, "should fail when the referenced user does not exist")
	})
}

func newTestPost(t *testing.T, userID uuid.UUID) *post.Post {
	t.Helper()

	return &post.Post{
		UserID:  userID,
		Content: "some random content",
	}
}

func newTestUser(ctx context.Context, t *testing.T) *auth.User {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	pool, err := database.NewPool(ctx, cfg.DB)
	require.NoError(t, err, "failed to create database pool")

	authRepo := auth.NewRepository(pool)

	user := &auth.User{
		Email:        fmt.Sprintf("test@email.com_%s", t.Name()),
		Username:     t.Name(),
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

func newTestRepository(t *testing.T) post.Repository {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	pool, err := database.NewPool(t.Context(), cfg.DB)
	require.NoError(t, err, "failed to create database pool")

	return post.NewPostgresRepository(pool)
}
