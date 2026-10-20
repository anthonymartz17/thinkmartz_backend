package feed

import (
	"errors"
)

var (
	// ErrInvalidCursor is returned when a client-supplied cursor can't be decoded.
	ErrInvalidCursor = errors.New("invalid cursor")
)
