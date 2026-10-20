package feed_test

import (
	"errors"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/feed"
	"github.com/anthonymartz17/thinkmartz_backend/internal/feed/mocks"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const testThreshold config.CelebrityFollowersThreshold = 1000

func TestGetFeed(t *testing.T) {
	ctx := t.Context()
	userID := uuid.New()

	t.Run("cursor present: repo error returns error, no cache calls", func(t *testing.T) {
		// arrange
		svc, _, mockRepo := newTestService(t)
		cursor := &feed.Cursor{PostID: uuid.New(), CreatedAt: time.Now()}
		repoErr := errors.New("db down")
		mockRepo.EXPECT().ListFolloweePostsBefore(ctx, userID, cursor, feed.FeedPageSize+1).Return(nil, repoErr)

		// act
		gotPosts, gotCursor, gotErr := svc.GetFeed(ctx, userID, cursor)

		// assert
		assert.Nil(t, gotPosts)
		assert.Nil(t, gotCursor)
		assert.ErrorIs(t, gotErr, repoErr)
	})

	t.Run("cursor present: last page returns nil cursor", func(t *testing.T) {
		// arrange
		svc, _, mockRepo := newTestService(t)
		cursor := &feed.Cursor{PostID: uuid.New(), CreatedAt: time.Now()}
		posts := makePosts(feed.FeedPageSize)
		mockRepo.EXPECT().ListFolloweePostsBefore(ctx, userID, cursor, feed.FeedPageSize+1).Return(posts, nil)

		// act
		gotPosts, gotCursor, gotErr := svc.GetFeed(ctx, userID, cursor)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, posts, gotPosts)
		assert.Nil(t, gotCursor)
	})

	t.Run("cursor present: more pages returns a cursor at the page boundary", func(t *testing.T) {
		// arrange
		svc, _, mockRepo := newTestService(t)
		cursor := &feed.Cursor{PostID: uuid.New(), CreatedAt: time.Now()}
		posts := makePosts(feed.FeedPageSize + 1)
		mockRepo.EXPECT().ListFolloweePostsBefore(ctx, userID, cursor, feed.FeedPageSize+1).Return(posts, nil)

		// act
		gotPosts, gotCursor, gotErr := svc.GetFeed(ctx, userID, cursor)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, posts[:feed.FeedPageSize], gotPosts)
		require.NotNil(t, gotCursor)
		last := posts[feed.FeedPageSize-1]
		assert.Equal(t, &feed.Cursor{PostID: last.ID, CreatedAt: last.CreatedAt}, gotCursor)
	})

	t.Run("no cursor, cache hit: uses cached ids without rebuilding", func(t *testing.T) {
		// arrange
		svc, mockCache, mockRepo := newTestService(t)
		ids := []uuid.UUID{uuid.New(), uuid.New()}
		posts := makePosts(2)
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(ids, nil)
		mockRepo.EXPECT().ListWithCelebrityPosts(ctx, userID, ids, int(testThreshold), feed.FeedPageSize+1).Return(posts, nil)

		// act
		gotPosts, gotCursor, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, posts, gotPosts)
		assert.Nil(t, gotCursor)
	})

	t.Run("no cursor, cache miss: rebuilds, sets cache, caps ids at a page", func(t *testing.T) {
		// arrange
		svc, mockCache, mockRepo := newTestService(t)
		entries := makeEntries(feed.FeedPageSize + 1)
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(nil, nil)
		mockRepo.EXPECT().ListNonCelebrityFeedEntries(ctx, userID, int(testThreshold), feed.FeedMaxSize).Return(entries, nil)
		mockCache.EXPECT().Set(ctx, userID, entries).Return(nil)
		wantIDs := entryIDs(entries)
		posts := makePosts(3)
		mockRepo.EXPECT().ListWithCelebrityPosts(ctx, userID, wantIDs, int(testThreshold), feed.FeedPageSize+1).Return(posts, nil)

		// act
		gotPosts, _, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, posts, gotPosts)
	})

	t.Run("no cursor, cache miss: more than a page of rebuilt entries (40) caps ids at a page", func(t *testing.T) {
		// arrange
		svc, mockCache, mockRepo := newTestService(t)
		entries := makeEntries(40)
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(nil, nil)
		mockRepo.EXPECT().ListNonCelebrityFeedEntries(ctx, userID, int(testThreshold), feed.FeedMaxSize).Return(entries, nil)
		mockCache.EXPECT().Set(ctx, userID, entries).Return(nil)
		wantIDs := entryIDs(entries[:feed.FeedPageSize+1])
		posts := makePosts(3)
		mockRepo.EXPECT().ListWithCelebrityPosts(ctx, userID, wantIDs, int(testThreshold), feed.FeedPageSize+1).Return(posts, nil)

		// act
		_, _, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		require.NoError(t, gotErr)
	})

	t.Run("no cursor, cache miss: follows nobody, merge still called with empty ids", func(t *testing.T) {
		// arrange
		svc, mockCache, mockRepo := newTestService(t)
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(nil, nil)
		mockRepo.EXPECT().ListNonCelebrityFeedEntries(ctx, userID, int(testThreshold), feed.FeedMaxSize).Return(nil, nil)
		mockCache.EXPECT().Set(ctx, userID, []feed.Entry(nil)).Return(nil)
		posts := makePosts(2)
		mockRepo.EXPECT().ListWithCelebrityPosts(ctx, userID, []uuid.UUID(nil), int(testThreshold), feed.FeedPageSize+1).Return(posts, nil)

		// act
		gotPosts, _, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, posts, gotPosts)
	})

	t.Run("no cursor, cache miss: Get error is treated as a miss", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache(ctrl)
		mockRepo := mocks.NewMockRepository(ctrl)
		core, logs := observer.New(zap.InfoLevel)
		svc := feed.NewService(mockCache, mockRepo, zap.New(core), testThreshold)

		getErr := errors.New("redis down")
		entries := makeEntries(1)
		posts := makePosts(1)
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(nil, getErr)
		mockRepo.EXPECT().ListNonCelebrityFeedEntries(ctx, userID, int(testThreshold), feed.FeedMaxSize).Return(entries, nil)
		mockCache.EXPECT().Set(ctx, userID, entries).Return(nil)
		mockRepo.EXPECT().ListWithCelebrityPosts(ctx, userID, entryIDs(entries), int(testThreshold), feed.FeedPageSize+1).Return(posts, nil)

		// act
		_, _, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, 1, logs.FilterMessage("get cached feed").Len())
	})

	t.Run("no cursor, cache miss: Set error doesn't fail the request", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache(ctrl)
		mockRepo := mocks.NewMockRepository(ctrl)
		core, logs := observer.New(zap.InfoLevel)
		svc := feed.NewService(mockCache, mockRepo, zap.New(core), testThreshold)

		setErr := errors.New("redis unreachable")
		entries := makeEntries(1)
		posts := makePosts(1)
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(nil, nil)
		mockRepo.EXPECT().ListNonCelebrityFeedEntries(ctx, userID, int(testThreshold), feed.FeedMaxSize).Return(entries, nil)
		mockCache.EXPECT().Set(ctx, userID, entries).Return(setErr)
		mockRepo.EXPECT().ListWithCelebrityPosts(ctx, userID, entryIDs(entries), int(testThreshold), feed.FeedPageSize+1).Return(posts, nil)

		// act
		_, _, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, 1, logs.FilterMessage("cache rebuild feed").Len())
	})

	t.Run("no cursor, cache miss: rebuild error returns error, merge never called", func(t *testing.T) {
		// arrange
		svc, mockCache, mockRepo := newTestService(t)
		rebuildErr := errors.New("postgres down")
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(nil, nil)
		mockRepo.EXPECT().ListNonCelebrityFeedEntries(ctx, userID, int(testThreshold), feed.FeedMaxSize).Return(nil, rebuildErr)

		// act
		gotPosts, gotCursor, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		assert.Nil(t, gotPosts)
		assert.Nil(t, gotCursor)
		assert.ErrorIs(t, gotErr, rebuildErr)
	})

	t.Run("merge: error is returned", func(t *testing.T) {
		// arrange
		svc, mockCache, mockRepo := newTestService(t)
		ids := []uuid.UUID{uuid.New()}
		mergeErr := errors.New("postgres down")
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(ids, nil)
		mockRepo.EXPECT().ListWithCelebrityPosts(ctx, userID, ids, int(testThreshold), feed.FeedPageSize+1).Return(nil, mergeErr)

		// act
		gotPosts, gotCursor, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		assert.Nil(t, gotPosts)
		assert.Nil(t, gotCursor)
		assert.ErrorIs(t, gotErr, mergeErr)
	})

	t.Run("merge: success with 21 rows is paginated", func(t *testing.T) {
		// arrange
		svc, mockCache, mockRepo := newTestService(t)
		ids := []uuid.UUID{uuid.New()}
		posts := makePosts(feed.FeedPageSize + 1)
		mockCache.EXPECT().Get(ctx, userID, feed.FeedPageSize+1).Return(ids, nil)
		mockRepo.EXPECT().ListWithCelebrityPosts(ctx, userID, ids, int(testThreshold), feed.FeedPageSize+1).Return(posts, nil)

		// act
		gotPosts, gotCursor, gotErr := svc.GetFeed(ctx, userID, nil)

		// assert
		require.NoError(t, gotErr)
		assert.Equal(t, posts[:feed.FeedPageSize], gotPosts)
		require.NotNil(t, gotCursor)
		last := posts[feed.FeedPageSize-1]
		assert.Equal(t, &feed.Cursor{PostID: last.ID, CreatedAt: last.CreatedAt}, gotCursor)
	})
}

func TestPaginate(t *testing.T) {
	t.Run("0 posts", func(t *testing.T) {
		// act
		gotPage, gotCursor := feed.Paginate(nil)

		// assert
		assert.Empty(t, gotPage)
		assert.Nil(t, gotCursor)
	})

	t.Run("exactly a page", func(t *testing.T) {
		// arrange
		posts := makePosts(feed.FeedPageSize)

		// act
		gotPage, gotCursor := feed.Paginate(posts)

		// assert
		assert.Equal(t, posts, gotPage)
		assert.Nil(t, gotCursor)
	})

	t.Run("one more than a page: cursor is the last post of the page, not the next one", func(t *testing.T) {
		// arrange
		posts := makePosts(feed.FeedPageSize + 1)

		// act
		gotPage, gotCursor := feed.Paginate(posts)

		// assert
		require.Len(t, gotPage, feed.FeedPageSize)
		assert.Equal(t, posts[:feed.FeedPageSize], gotPage)
		require.NotNil(t, gotCursor)
		last := posts[feed.FeedPageSize-1]
		assert.Equal(t, &feed.Cursor{PostID: last.ID, CreatedAt: last.CreatedAt}, gotCursor)
	})
}

func newTestService(t *testing.T) (*feed.Service, *mocks.MockCache, *mocks.MockRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockCache := mocks.NewMockCache(ctrl)
	mockRepo := mocks.NewMockRepository(ctrl)
	svc := feed.NewService(mockCache, mockRepo, zap.NewNop(), testThreshold)
	return svc, mockCache, mockRepo
}

// makePosts builds n posts with distinct IDs, newest first.
func makePosts(n int) []post.Post {
	posts := make([]post.Post, n)
	now := time.Now()
	for i := range posts {
		posts[i] = post.Post{
			ID:        uuid.New(),
			CreatedAt: now.Add(-time.Duration(i) * time.Second),
		}
	}
	return posts
}

// makeEntries builds n feed entries with distinct IDs, newest first.
func makeEntries(n int) []feed.Entry {
	entries := make([]feed.Entry, n)
	now := time.Now()
	for i := range entries {
		entries[i] = feed.Entry{
			PostID:    uuid.New(),
			CreatedAt: now.Add(-time.Duration(i) * time.Second),
		}
	}
	return entries
}

func entryIDs(entries []feed.Entry) []uuid.UUID {
	ids := make([]uuid.UUID, len(entries))
	for i, e := range entries {
		ids[i] = e.PostID
	}
	return ids
}
