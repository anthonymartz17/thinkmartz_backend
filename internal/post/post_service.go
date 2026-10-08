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

// UpdateInput holds the parameters for Service.Update.
type UpdateInput struct {
	PostID  uuid.UUID
	UserID  uuid.UUID
	Content string
}

// DeleteInput identifies the post to delete and the caller deleting it.
type DeleteInput struct {
	PostID uuid.UUID
	UserID uuid.UUID
}

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

// GetByID gets a single post by ID
func (s *service) GetByID(ctx context.Context, postID uuid.UUID) (*Post, error) {
	return s.repo.GetByID(ctx, postID)
}

// Update sets the content of the post with input.postID. It returns ErrPostNotFound
// if the post doesn't exist and ErrForbidden if input.userID doesn't own it.
func (s *service) Update(ctx context.Context, input UpdateInput) (*Post, error) {

	post, err := s.repo.GetByID(ctx, input.PostID)
	if err != nil {
		return nil, fmt.Errorf("update post: %w", err)
	}

	if post.UserID != input.UserID {
		return nil, ErrForbidden
	}

	post.Content = input.Content
	if err := s.repo.Update(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

// Delete removes a post owned by the caller and, best-effort, removes it
// from followers' feeds. Returns ErrPostNotFound or ErrForbidden.
func (s *service) Delete(ctx context.Context, input DeleteInput) error {
	post, err := s.repo.GetByID(ctx, input.PostID)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}

	if post.UserID != input.UserID {
		return ErrForbidden
	}

	if err := s.repo.Delete(ctx, post.ID); err != nil {
		return err
	}

	// Best-effort feed cleanup: the post is gone from Postgres, so every
	// failure below is logged and swallowed.
	count, err := s.repo.CountFollowers(ctx, post.UserID)
	if err != nil {
		s.logger.Warn("count followers for feed removal",
			zap.String("post_id", post.ID.String()),
			zap.Error(err),
		)
		return nil
	}

	if count >= int(s.CelebrityFollowersThreshold) {
		return nil // celebrity posts are never fanned out
	}

	followerIDs, err := s.repo.GetFollowersByID(ctx, post.UserID)
	if err != nil {
		s.logger.Warn("get followers for feed removal",
			zap.String("post_id", post.ID.String()),
			zap.Error(err),
		)
		return nil
	}

	if err := s.feedRepo.RemoveFromFeeds(ctx, post.ID, followerIDs); err != nil {
		s.logger.Warn("remove post from feeds",
			zap.String("post_id", post.ID.String()),
			zap.Error(err),
		)
	}

	return nil
}
