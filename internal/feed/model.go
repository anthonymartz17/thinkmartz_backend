package feed

import (
	"time"

	"github.com/google/uuid"
)

// Entry is a single post's position in a feed timeline: its ID and the
// CreatedAt it's scored by. Used both as feed content (Cache.Set) and, once
// ListPostsBefore exists, as the pagination marker for the next page.
type Entry struct {
	PostID    uuid.UUID
	CreatedAt time.Time
}
