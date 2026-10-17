package feed_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/database"
	"github.com/anthonymartz17/thinkmartz_backend/internal/feed"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFolloweePostsBefore(t *testing.T) {
	t.Run("returns only followees' posts, newest first", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		followee := newTestUser(ctx, t, pool)
		stranger := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, followee.ID)

		now := time.Now()
		older := seedPost(ctx, t, pool, followee.ID, now.Add(-time.Hour))
		newer := seedPost(ctx, t, pool, followee.ID, now)
		seedPost(ctx, t, pool, stranger.ID, now)

		cursor := &feed.Entry{CreatedAt: now.Add(time.Hour), PostID: uuid.New()}

		// act
		gotPosts, gotErr := repo.ListFolloweePostsBefore(ctx, userA.ID, cursor, 20)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotPosts, 2, "should only include the followee's posts")
		assert.Equal(t, newer, gotPosts[0].ID, "newest post should come first")
		assert.Equal(t, older, gotPosts[1].ID)
	})

	t.Run("respects the cursor", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		followee := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, followee.ID)

		t0 := time.Now()
		oldest := seedPost(ctx, t, pool, followee.ID, t0)
		middle := seedPost(ctx, t, pool, followee.ID, t0.Add(time.Second))
		newest := seedPost(ctx, t, pool, followee.ID, t0.Add(2*time.Second))

		cursor := &feed.Entry{CreatedAt: t0.Add(2 * time.Second), PostID: newest}

		// act
		gotPosts, gotErr := repo.ListFolloweePostsBefore(ctx, userA.ID, cursor, 20)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotPosts, 2, "should exclude the cursor's own post and anything newer")
		assert.Equal(t, middle, gotPosts[0].ID)
		assert.Equal(t, oldest, gotPosts[1].ID)
	})

	t.Run("tie-break on equal created_at uses id", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		followee := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, followee.ID)

		sameTime := time.Now()
		first := seedPost(ctx, t, pool, followee.ID, sameTime)
		second := seedPost(ctx, t, pool, followee.ID, sameTime)
		higherID, lowerID := first, second
		if second.String() > first.String() {
			higherID, lowerID = second, first
		}

		cursor := &feed.Entry{CreatedAt: sameTime, PostID: higherID}

		// act
		gotPosts, gotErr := repo.ListFolloweePostsBefore(ctx, userA.ID, cursor, 20)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotPosts, 1, "only the lower-id post at the same timestamp should be strictly before the cursor")
		assert.Equal(t, lowerID, gotPosts[0].ID)
	})

	t.Run("respects the limit", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		followee := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, followee.ID)

		now := time.Now()
		var want []uuid.UUID
		for i := 0; i < 5; i++ {
			id := seedPost(ctx, t, pool, followee.ID, now.Add(time.Duration(i)*time.Second))
			want = append([]uuid.UUID{id}, want...) // newest first
		}
		cursor := &feed.Entry{CreatedAt: now.Add(time.Hour), PostID: uuid.New()}

		// act
		gotPosts, gotErr := repo.ListFolloweePostsBefore(ctx, userA.ID, cursor, 2)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotPosts, 2, "should cap at the limit")
		assert.Equal(t, want[0], gotPosts[0].ID)
		assert.Equal(t, want[1], gotPosts[1].ID)
	})

	t.Run("no followees returns empty slice, nil error", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		cursor := &feed.Entry{CreatedAt: time.Now(), PostID: uuid.New()}

		// act
		gotPosts, gotErr := repo.ListFolloweePostsBefore(ctx, userA.ID, cursor, 20)

		// assert
		require.NoError(t, gotErr)
		assert.Empty(t, gotPosts)
	})
}

func TestListNonCelebrityFeedEntries(t *testing.T) {
	const threshold = 1000

	t.Run("excludes followees at or above the threshold", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		celeb := newTestUser(ctx, t, pool)
		normal := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, celeb.ID)
		seedFollow(ctx, t, pool, userA.ID, normal.ID)
		setFollowerCount(ctx, t, pool, celeb.ID, threshold)
		setFollowerCount(ctx, t, pool, normal.ID, threshold-1)

		now := time.Now()
		seedPost(ctx, t, pool, celeb.ID, now)
		normalPost := seedPost(ctx, t, pool, normal.ID, now)

		// act
		gotEntries, gotErr := repo.ListNonCelebrityFeedEntries(ctx, userA.ID, threshold, 20)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotEntries, 1, "a follower_count exactly at the threshold should be excluded")
		assert.Equal(t, normalPost, gotEntries[0].PostID)
	})

	t.Run("returns entries newest first", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		normal := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, normal.ID)
		setFollowerCount(ctx, t, pool, normal.ID, 0)

		now := time.Now()
		older := seedPost(ctx, t, pool, normal.ID, now.Add(-time.Hour))
		newer := seedPost(ctx, t, pool, normal.ID, now)

		// act
		gotEntries, gotErr := repo.ListNonCelebrityFeedEntries(ctx, userA.ID, threshold, 20)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotEntries, 2)
		assert.Equal(t, newer, gotEntries[0].PostID)
		assert.Equal(t, older, gotEntries[1].PostID)
	})

	t.Run("respects the limit", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		normal := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, normal.ID)
		setFollowerCount(ctx, t, pool, normal.ID, 0)

		now := time.Now()
		for i := 0; i < 5; i++ {
			seedPost(ctx, t, pool, normal.ID, now.Add(time.Duration(i)*time.Second))
		}

		// act
		gotEntries, gotErr := repo.ListNonCelebrityFeedEntries(ctx, userA.ID, threshold, 2)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotEntries, 2, "should cap at the limit")
	})

	t.Run("no qualifying followees returns empty slice, nil error", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)

		// act
		gotEntries, gotErr := repo.ListNonCelebrityFeedEntries(ctx, userA.ID, threshold, 20)

		// assert
		require.NoError(t, gotErr)
		assert.Empty(t, gotEntries)
	})
}

func TestListWithCelebrityPosts(t *testing.T) {
	const threshold = 1000

	t.Run("merges explicit ids with celebrity followees' posts", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		celeb := newTestUser(ctx, t, pool)
		stranger := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, celeb.ID)
		setFollowerCount(ctx, t, pool, celeb.ID, threshold)

		now := time.Now()
		celebPost := seedPost(ctx, t, pool, celeb.ID, now)
		strangerPost := seedPost(ctx, t, pool, stranger.ID, now.Add(-time.Hour))

		// act
		gotPosts, gotErr := repo.ListWithCelebrityPosts(ctx, userA.ID, []uuid.UUID{strangerPost}, threshold, 20)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotPosts, 2)
		assert.Equal(t, celebPost, gotPosts[0].ID, "newest should come first")
		assert.Equal(t, strangerPost, gotPosts[1].ID)
	})

	t.Run("boundary: follower_count == threshold is included", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		celeb := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, celeb.ID)
		setFollowerCount(ctx, t, pool, celeb.ID, threshold)

		celebPost := seedPost(ctx, t, pool, celeb.ID, time.Now())

		// act
		gotPosts, gotErr := repo.ListWithCelebrityPosts(ctx, userA.ID, nil, threshold, 20)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotPosts, 1)
		assert.Equal(t, celebPost, gotPosts[0].ID)
	})

	t.Run("excludes followees below the threshold unless named in ids", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		normal := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, normal.ID)
		setFollowerCount(ctx, t, pool, normal.ID, threshold-1)
		seedPost(ctx, t, pool, normal.ID, time.Now())

		// act
		gotPosts, gotErr := repo.ListWithCelebrityPosts(ctx, userA.ID, nil, threshold, 20)

		// assert
		require.NoError(t, gotErr)
		assert.Empty(t, gotPosts, "a below-threshold followee's post should not appear unless named in ids")
	})

	t.Run("respects the limit on the merged result", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		celeb := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, celeb.ID)
		setFollowerCount(ctx, t, pool, celeb.ID, threshold)

		now := time.Now()
		var ids []uuid.UUID
		for i := 0; i < 3; i++ {
			ids = append(ids, seedPost(ctx, t, pool, celeb.ID, now.Add(time.Duration(i)*time.Second)))
		}

		// act
		gotPosts, gotErr := repo.ListWithCelebrityPosts(ctx, userA.ID, ids, threshold, 2)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotPosts, 2, "should cap the merged result at the limit")
	})

	t.Run("empty ids still returns celebrity posts", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		pool := newTestPool(t)
		repo := feed.NewPostgresRepository(pool)
		userA := newTestUser(ctx, t, pool)
		celeb := newTestUser(ctx, t, pool)
		seedFollow(ctx, t, pool, userA.ID, celeb.ID)
		setFollowerCount(ctx, t, pool, celeb.ID, threshold)
		celebPost := seedPost(ctx, t, pool, celeb.ID, time.Now())

		// act
		gotPosts, gotErr := repo.ListWithCelebrityPosts(ctx, userA.ID, nil, threshold, 20)

		// assert
		require.NoError(t, gotErr)
		require.Len(t, gotPosts, 1)
		assert.Equal(t, celebPost, gotPosts[0].ID)
	})
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "configuration failed to load")

	pool, err := database.NewPool(t.Context(), cfg.DB)
	require.NoError(t, err, "failed to create database pool")

	return pool
}

func newTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool) *auth.User {
	t.Helper()

	authRepo := auth.NewPostgresRepository(pool)

	user := &auth.User{
		Email:        fmt.Sprintf("test-%s@email.com", uuid.New().String()),
		Username:     fmt.Sprintf("%s-%s", t.Name(), uuid.New().String()[:8]),
		PasswordHash: "fake-hashed-password",
	}
	require.NoError(t, authRepo.Save(ctx, user), "failed to seed test user")

	t.Cleanup(func() {
		// ON DELETE CASCADE on posts.user_id and follows' FKs also removes any
		// posts/follows seeded against this user, so no separate cleanup is needed.
		_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
		if err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	})

	return user
}

// seedPost inserts a post with an explicit id and created_at, bypassing
// post.Repository.Save (which always uses now()), so tests can control
// ordering precisely.
func seedPost(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, createdAt time.Time) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO posts (id, user_id, content, created_at) VALUES ($1, $2, 'x', $3)`,
		id, userID, createdAt)
	require.NoError(t, err, "failed to seed post")
	return id
}

func seedFollow(ctx context.Context, t *testing.T, pool *pgxpool.Pool, followerID, followeeID uuid.UUID) {
	t.Helper()

	_, err := pool.Exec(ctx,
		`INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2)`, followerID, followeeID)
	require.NoError(t, err, "failed to seed follow")
}

// setFollowerCount overwrites the trigger-maintained follower_count directly,
// so celebrity-threshold tests don't need real follow rows to reach the count.
func setFollowerCount(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, count int) {
	t.Helper()

	_, err := pool.Exec(ctx, `UPDATE users SET follower_count = $1 WHERE id = $2`, count, userID)
	require.NoError(t, err, "failed to set follower_count")
}
