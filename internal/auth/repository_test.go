package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Save_Success(t *testing.T) {
	// 1. Arrange: set up the real dependency

	repo := newTestRepository(t)
	user := newTestUser(t)

	// 2. Act: call the method you're testing
	gotErr := repo.Save(t.Context(), user)

	// 3. Assert: check the result is what you expect
	assert.NoError(t, gotErr, "gotErr should be nil")
	assert.NotEqual(t, uuid.Nil, user.ID, "ID should not be UUID zero value")
	assert.False(t, user.CreatedAt.IsZero(), "Timestamp should not be zero")

	t.Cleanup(func() {
		_, err := repo.Pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
		if err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	})
}

func TestRepository_Save_Duplicate(t *testing.T) {
	//arrange
	repo := newTestRepository(t)
	user := newTestUser(t)
	ctx := t.Context()
	firstErr := repo.Save(ctx, user)
	require.NoError(t, firstErr, "First save should succeed")

	//act
	gotErr := repo.Save(ctx, user)

	//assert
	assert.ErrorIs(t, gotErr, ErrAlreadyExists, "error should equal ErrAlreadyExists")

	t.Cleanup(func() {
		_, err := repo.Pool.Exec(context.Background(), `DELETE FROM users WHERE ID = $1`, user.ID)

		if err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	})
}

func TestFindByEmail_Success(t *testing.T) {
	// arrange
	ctx := t.Context()
	repo := newTestRepository(t)
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
		_, err := repo.Pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, gotUser.ID)

		if err != nil {
			t.Logf("cleanup failed: %v", err)

		}
	})

}

func TestFindByEmail_NotFound(t *testing.T) {
	// arrange
	ctx := t.Context()
	repo := newTestRepository(t)

	// act
	gotUser, gotErr := repo.FindByEmail(ctx, "test@email.com")
	// assert
	assert.Nil(t, gotUser, "expected nil for gotUser")
	assert.Error(t, gotErr, "expected a not found error for non-existing user")

}

func newTestUser(t *testing.T) *User {
	t.Helper()

	user := &User{
		Email:        fmt.Sprintf("test@email.com_%s", t.Name()),
		Username:     t.Name(),
		PasswordHash: "fake-hashed-password",
	}
	return user
}

func newTestRepository(t *testing.T) *Repository {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	pool, err := database.NewPool(t.Context(), cfg)
	require.NoError(t, err, "Failed to create database pool")

	return NewRepository(pool)
}
