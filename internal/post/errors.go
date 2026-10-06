package post

import "errors"

var (
	// ErrPostNotFound is returned when no post matches the given userID.
	ErrPostNotFound = errors.New("post not found")
	// ErrForbidden is returned when an authenticated user attempts to modify a post they don't own.
	ErrForbidden = errors.New("forbidden")
)
