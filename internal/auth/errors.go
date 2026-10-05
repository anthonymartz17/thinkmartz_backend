package auth

import "errors"

var (
	// ErrInvalidPassword indicates the provided password did not match the stored hash.
	ErrInvalidPassword = errors.New("password is invalid")
	// ErrAlreadyExists is returned when a user's email or username is
	// already taken.
	ErrAlreadyExists = errors.New("already exists")
	// ErrEmailAlreadyExists is returned when a user's email is already taken.
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrUsernameAlreadyExists is returned when a username is already taken.
	ErrUsernameAlreadyExists = errors.New("username already exists")
	// ErrUserNotFound is returned when no user matches the given email.
	ErrUserNotFound = errors.New("user not found")
	// ErrRefreshTokenNotFound is returned when no match for refresh token is not found on redis
	ErrRefreshTokenNotFound = errors.New("token not found")
)
