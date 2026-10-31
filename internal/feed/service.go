package feed

import (
	"context"
	"fmt"

	"github.com/anthonymartz17/thinkmartz_backend/internal/config"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// _feedPageSize is how many posts GET /feed returns per page (per the API spec).
	_feedPageSize = 20
	// _feedMaxSize keeps two pages of IDs so page one stays full even if
	// hydration deletes a few posts.
	_feedMaxSize = 2 * _feedPageSize
)

// validates Service implements Reader
var _ Reader = (*Service)(nil)

// Service assembles a user's feed from the Redis cache and, on a cache
// miss or for older pages, from Postgres.
type Service struct {
	cache     Cache
	repo      Repository
	logger    *zap.Logger
	threshold int
}

// NewService builds a Service from its cache, repository, logger, and
// celebrity follower-count threshold.
func NewService(c Cache, r Repository, l *zap.Logger, t config.CelebrityFollowersThreshold) *Service {
	return &Service{
		cache:     c,
		repo:      r,
		logger:    l,
		threshold: int(t),
	}
}

// GetFeed returns a page of a user's feed and a cursor for the next page
// (nil if this was the last page). With a cursor, it reads further back
// through the user's followees' posts. Without one, it serves the first
// page from the cache (rebuilding it from Postgres on a miss) merged with
// celebrity followees' posts, which are never cached.
func (s *Service) GetFeed(ctx context.Context, userID uuid.UUID, cursor *Cursor) ([]post.Post, *Cursor, error) {

	if cursor != nil {
		posts, err := s.repo.ListFolloweePostsBefore(ctx, userID, cursor, _feedPageSize+1)
		if err != nil {
			return nil, nil, fmt.Errorf("list older posts: %w", err)
		}

		page, next := paginate(posts)

		return page, next, nil
	}

	ids, err := s.firstPageIDs(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	posts, err := s.repo.ListWithCelebrityPosts(ctx, userID, ids, s.threshold, _feedPageSize+1)

	if err != nil {
		return nil, nil, fmt.Errorf("list first feed page: %w", err)
	}
	page, next := paginate(posts)

	return page, next, nil

}

func (s *Service) firstPageIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.cache.Get(ctx, userID, _feedPageSize+1)

	if err != nil {
		s.logger.Error("get cached feed", zap.Error(err), zap.Stringer("userID", userID))
	}

	if len(ids) == 0 {
		entries, err := s.repo.ListNonCelebrityFeedEntries(ctx, userID, s.threshold, _feedMaxSize)

		if err != nil {
			return nil, fmt.Errorf("feed rebuild: %w", err)
		}

		if err := s.cache.Set(ctx, userID, entries); err != nil {
			s.logger.Error("cache rebuild feed",
				zap.Error(err),
				zap.Stringer("userID", userID),
			)
		}
		n := min(len(entries), _feedPageSize+1)
		for _, e := range entries[:n] {
			ids = append(ids, e.PostID)
		}

	}

	return ids, nil
}

func paginate(posts []post.Post) ([]post.Post, *Cursor) {
	if len(posts) <= _feedPageSize {
		return posts, nil
	}

	page := posts[:_feedPageSize]
	last := posts[len(page)-1]
	next := &Cursor{
		PostID:    last.ID,
		CreatedAt: last.CreatedAt,
	}

	return page, next

}
