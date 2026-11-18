package user

import "context"

// Repository persists user relationships.
type Repository interface {
	// SaveFollow records that f.FollowerID follows f.FolloweeID.
	// Saving an existing follow is a no-op. Returns ErrUserNotFound
	// if the followee does not exist.
	SaveFollow(ctx context.Context, f Follow) error
}
