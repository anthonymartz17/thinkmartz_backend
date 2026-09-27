package post

import (
	"time"

	"github.com/google/uuid"
)

// Post is post's minimal view of a user row.
type Post struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Content      string
	LikeCount    int
	CommentCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
