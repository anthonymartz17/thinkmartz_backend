package post

import (
	"context"
	"fmt"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// validates service implements Service
var _ Service = (*service)(nil)

// service handles post business logic orchestrating interaction with Repository and RedisRepository
type service struct {
	repo                        Repository
	feedRepo                    FeedRepository
	CelebrityFollowersThreshold config.CelebrityFollowersThreshold
	logger                      *zap.Logger
}

// NewService creates a new instance of Service
func NewService(repo Repository, feedRepo FeedRepository, threshold config.CelebrityFollowersThreshold, logger *zap.Logger) Service {
	return &service{
		repo:                        repo,
		feedRepo:                    feedRepo,
		CelebrityFollowersThreshold: threshold,
		logger:                      logger,
	}
}

// Create saves a new post and fans it out to the author's followers' feeds,
// unless the author is a celebrity account (fanning out synchronously to a
// huge follower count is too expensive to do here). Fan-out failures are
// logged but do not fail the overall call, since the post itself was already
// successfully created by that point.
func (s *service) Create(ctx context.Context, userID uuid.UUID, content string) (*Post, error) {
	post := &Post{
		UserID:  userID,
		Content: content,
	}

	if err := s.repo.Save(ctx, post); err != nil {
		return nil, fmt.Errorf("save post: %w", err)
	}

	count, err := s.repo.CountFollowers(ctx, userID)
	if err != nil {
		s.logger.Error("count followers after post creation", zap.Error(err), zap.Stringer("postID", post.ID))
		return post, nil
	}

	if count >= int(s.CelebrityFollowersThreshold) {
		return post, nil
	}

	followerIDs, err := s.repo.GetFollowersByID(ctx, userID)
	if err != nil {
		s.logger.Error("get followers for feed fan-out", zap.Error(err), zap.Stringer("postID", post.ID))
		return post, nil
	}

	if err := s.feedRepo.AddToFeed(ctx, *post, followerIDs); err != nil {
		s.logger.Error("fan out post to feeds", zap.Error(err), zap.Stringer("postID", post.ID))
		return post, nil
	}

	return post, nil
}
