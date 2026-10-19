package feed

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Entry is a single post's position in a feed timeline: its ID and the
// CreatedAt it's scored by. Used both as feed content (Cache.Set) and, once
// ListPostsBefore exists, as the pagination marker for the next page.
type Entry struct {
	PostID    uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// Cursor marks a position in a feed: the next page starts strictly after
// this post in (created_at, id) order.
type Cursor struct {
	PostID    uuid.UUID
	CreatedAt time.Time
}

// Encode returns the cursor as an opaque, URL-safe string.
func (c Cursor) Encode() string {
	raw := strconv.FormatInt(c.CreatedAt.UnixMicro(), 10) + ":" + c.PostID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a string produced by Cursor.Encode. Any malformed
// input returns an error wrapping ErrInvalidCursor.
func DecodeCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: decode base64: %v", ErrInvalidCursor, err)
	}

	micros, id, ok := strings.Cut(string(raw), ":")
	if !ok {
		return Cursor{}, fmt.Errorf("%w: missing separator", ErrInvalidCursor)
	}

	n, err := strconv.ParseInt(micros, 10, 64)
	if err != nil || n <= 0 {
		return Cursor{}, fmt.Errorf("%w: bad timestamp", ErrInvalidCursor)
	}

	postID, err := uuid.Parse(id)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: bad post id: %v", ErrInvalidCursor, err)
	}

	return Cursor{CreatedAt: time.UnixMicro(n).UTC(), PostID: postID}, nil
}
