package auth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Save(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		// 1. Arrange: set up the real dependency
		pool := newTestPool(t)
		repo := auth.NewPostgresRepository(pool)
		user := newTestUser(t)

		// 2. Act: call the method you're testing
		gotErr := repo.Save(t.Context(), user)

		// 3. Assert: check the result is what you expect
		assert.NoError(t, gotErr, "gotErr should be nil")
		assert.NotEqual(t, uuid.Nil, user.ID, "ID should not be UUID zero value")
		assert.False(t, user.CreatedAt.IsZero(), "Timestamp should not be zero")

		t.Cleanup(func() {
			_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
			if err != nil {
				t.Logf("cleanup failed: %v", err)
			}
		})
	})
	t.Run("duplicate email", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		repo := auth.NewPostgresRepository(pool)
		ctx := t.Context()
		first := newTestUser(t)
		firstErr := repo.Save(ctx, first)
		require.NoError(t, firstErr, "First save should succeed")

		second := &auth.User{
			Email:        first.Email,
			Username:     first.Username + "-second",
			PasswordHash: "fake-hashed-password",
		}

		// act
		gotErr := repo.Save(ctx, second)

		// assert
		assert.ErrorIs(t, gotErr, auth.ErrEmailAlreadyExists, "error should equal ErrEmailAlreadyExists")

		t.Cleanup(func() {
			_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, first.ID)

			if err != nil {
				t.Logf("cleanup failed: %v", err)
			}
		})
	})

	t.Run("duplicate username", func(t *testing.T) {
		// arrange
		pool := newTestPool(t)
		repo := auth.NewPostgresRepository(pool)
		ctx := t.Context()
		first := newTestUser(t)
		firstErr := repo.Save(ctx, first)
		require.NoError(t, firstErr, "First save should succeed")

		second := &auth.User{
			Email:        "second-" + first.Email,
			Username:     first.Username,
			PasswordHash: "fake-hashed-password",
		}

		// act
		gotErr := repo.Save(ctx, second)

		// assert
		assert.ErrorIs(t, gotErr, auth.ErrUsernameAlreadyExists, "error should equal ErrUsernameAlreadyExists")

		t.Cleanup(func() {
			_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, first.ID)

			if err != nil {
				t.Logf("cleanup failed: %v", err)
			}
		})
	})
}

func TestFindByEmail(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := auth.NewPostgresRepository(pool)
		user := newTestUser(t)

		err := repo.Save(ctx, user)
		require.NoError(t, err, "should not fail setting up fake user")

		// act
		gotUser, gotErr := repo.FindByEmail(ctx, user.Email)

		// assert
		assert.NoError(t, gotErr, "should not fail to find an exisiting user by email")
		assert.Equal(t, user.ID, gotUser.ID, "should match expected user ID")
		assert.Equal(t, user.Email, gotUser.Email, "should match expected user Email")
		assert.Equal(t, user.PasswordHash, gotUser.PasswordHash, "should match expected user PasswordHash")
		assert.Equal(t, user.Username, gotUser.Username, "should match expected user Username")
		assert.Equal(t, user.CreatedAt, gotUser.CreatedAt, "should match expected user CreatedAt")

		t.Cleanup(func() {
			_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, gotUser.ID)

			if err != nil {
				t.Logf("cleanup failed: %v", err)

			}
		})
	})
	t.Run("Not found", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := auth.NewPostgresRepository(pool)

		// act
		gotUser, gotErr := repo.FindByEmail(ctx, "test@email.com")
		// assert
		assert.Nil(t, gotUser, "expected nil for gotUser")
		assert.Error(t, gotErr, "expected a not found error for non-existing user")
	})
}

func TestDelete(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := auth.NewPostgresRepository(pool)
		user := newTestUser(t)

		err := repo.Save(ctx, user)
		require.NoError(t, err, "should not fail to save fake user")

		// act
		gotErr := repo.Delete(ctx, user.ID)
		_, gotFindByEmailErr := repo.FindByEmail(ctx, user.Email)

		// assert
		assert.NoError(t, gotErr, "should delete existing user successfully")
		assert.ErrorIs(t, gotFindByEmailErr, auth.ErrUserNotFound, "Expected ErrUserNotFound after successful deletion")
	})

	t.Run("Not found", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := auth.NewPostgresRepository(pool)
		nonExistentID := uuid.New()

		// act
		gotErr := repo.Delete(ctx, nonExistentID)

		// assert
		assert.ErrorIs(t, gotErr, auth.ErrUserNotFound, "expected ErrUserNotFound specifically")
	})

}

func newTestUser(t *testing.T) *auth.User {
	t.Helper()

	user := &auth.User{
		Email:        fmt.Sprintf("test@email.com_%s", t.Name()),
		Username:     t.Name(),
		PasswordHash: "fake-hashed-password",
	}
	return user
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	pool, err := database.NewPool(t.Context(), cfg.DB)
	require.NoError(t, err, "Failed to create database pool")

	return pool
}
