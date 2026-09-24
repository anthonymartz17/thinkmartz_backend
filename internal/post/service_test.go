package post_test

import (
	"errors"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestService_Create(t *testing.T) {
	ctx := t.Context()
	userID := uuid.New()
	content := "hello world"
	saveErr := errors.New("save failed")
	countErr := errors.New("count followers failed")
	followersErr := errors.New("get followers failed")
	feedErr := errors.New("add to feed failed")

	t.Run("save fails", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockRepository(ctrl)
		mockFeedRepo := mocks.NewMockFeedRepository(ctrl)
		logger := zap.NewNop()
		const celebrityFollowersThreshold config.CelebrityFollowersThreshold = 10
		svc := post.NewService(mockRepo, mockFeedRepo, celebrityFollowersThreshold, logger)

		mockRepo.EXPECT().Save(ctx, &post.Post{UserID: userID, Content: content}).Return(saveErr)

		// act
		gotPost, gotErr := svc.Create(ctx, userID, content)

		//assert
		assert.Nil(t, gotPost)
		assert.ErrorIs(t, gotErr, saveErr)
	})

	t.Run("count followers fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockRepository(ctrl)
		mockFeedRepo := mocks.NewMockFeedRepository(ctrl)
		core, logs := observer.New(zap.InfoLevel)
		logger := zap.New(core)
		const celebrityFollowersThreshold config.CelebrityFollowersThreshold = 10

		svc := post.NewService(mockRepo, mockFeedRepo, celebrityFollowersThreshold, logger)

		mockRepo.EXPECT().Save(ctx, &post.Post{UserID: userID, Content: content}).Return(nil)
		mockRepo.EXPECT().CountFollowers(ctx, userID).Return(0, countErr)

		gotPost, gotErr := svc.Create(ctx, userID, content)

		require.NoError(t, gotErr)
		require.NotNil(t, gotPost)
		assert.Equal(t, 1, logs.FilterMessage("count followers after post creation").Len())
	})

	t.Run("celebrity threshold met, no fan-out", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockRepository(ctrl)
		mockFeedRepo := mocks.NewMockFeedRepository(ctrl)
		logger := zap.NewNop()
		const celebrityFollowersThreshold config.CelebrityFollowersThreshold = 10

		svc := post.NewService(mockRepo, mockFeedRepo, celebrityFollowersThreshold, logger)

		mockRepo.EXPECT().Save(ctx, &post.Post{UserID: userID, Content: content}).Return(nil)
		mockRepo.EXPECT().CountFollowers(ctx, userID).Return(int(celebrityFollowersThreshold), nil)

		gotPost, gotErr := svc.Create(ctx, userID, content)

		require.NoError(t, gotErr)
		require.NotNil(t, gotPost)
	})

	t.Run("get followers fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockRepository(ctrl)
		mockFeedRepo := mocks.NewMockFeedRepository(ctrl)
		core, logs := observer.New(zap.InfoLevel)
		logger := zap.New(core)
		const celebrityFollowersThreshold config.CelebrityFollowersThreshold = 10

		svc := post.NewService(mockRepo, mockFeedRepo, celebrityFollowersThreshold, logger)

		mockRepo.EXPECT().Save(ctx, &post.Post{UserID: userID, Content: content}).Return(nil)
		mockRepo.EXPECT().CountFollowers(ctx, userID).Return(int(celebrityFollowersThreshold)-1, nil)
		mockRepo.EXPECT().GetFollowersByID(ctx, userID).Return(nil, followersErr)

		gotPost, gotErr := svc.Create(ctx, userID, content)

		require.NoError(t, gotErr)
		require.NotNil(t, gotPost)
		assert.Equal(t, 1, logs.FilterMessage("get followers for feed fan-out").Len())
	})

	t.Run("add to feed fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockRepository(ctrl)
		mockFeedRepo := mocks.NewMockFeedRepository(ctrl)
		core, logs := observer.New(zap.InfoLevel)
		logger := zap.New(core)
		const celebrityFollowersThreshold config.CelebrityFollowersThreshold = 10

		svc := post.NewService(mockRepo, mockFeedRepo, celebrityFollowersThreshold, logger)

		followerIDs := []uuid.UUID{uuid.New()}

		mockRepo.EXPECT().Save(ctx, &post.Post{UserID: userID, Content: content}).Return(nil)
		mockRepo.EXPECT().CountFollowers(ctx, userID).Return(int(celebrityFollowersThreshold)-1, nil)
		mockRepo.EXPECT().GetFollowersByID(ctx, userID).Return(followerIDs, nil)
		mockFeedRepo.EXPECT().AddToFeed(ctx, gomock.Any(), followerIDs).Return(feedErr)

		gotPost, gotErr := svc.Create(ctx, userID, content)

		require.NoError(t, gotErr)
		require.NotNil(t, gotPost)
		assert.Equal(t, 1, logs.FilterMessage("fan out post to feeds").Len())
	})

	t.Run("success, non-celebrity fan-out", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockRepository(ctrl)
		mockFeedRepo := mocks.NewMockFeedRepository(ctrl)
		logger := zap.NewNop()
		const celebrityFollowersThreshold config.CelebrityFollowersThreshold = 10

		svc := post.NewService(mockRepo, mockFeedRepo, celebrityFollowersThreshold, logger)

		followerIDs := []uuid.UUID{uuid.New()}

		mockRepo.EXPECT().Save(ctx, &post.Post{UserID: userID, Content: content}).Return(nil)
		mockRepo.EXPECT().CountFollowers(ctx, userID).Return(int(celebrityFollowersThreshold)-1, nil)
		mockRepo.EXPECT().GetFollowersByID(ctx, userID).Return(followerIDs, nil)
		mockFeedRepo.EXPECT().AddToFeed(ctx, gomock.Any(), followerIDs).Return(nil)

		gotPost, gotErr := svc.Create(ctx, userID, content)

		require.NoError(t, gotErr)
		require.NotNil(t, gotPost)
	})
}
