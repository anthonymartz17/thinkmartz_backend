package user

import "github.com/google/uuid"

// Follow is a row in the follows table: FollowerID follows FolloweeID.
type Follow struct {
	FollowerID uuid.UUID
	FolloweeID uuid.UUID
}
