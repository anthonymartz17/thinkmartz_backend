// Package user manages user relationships such as follows.
package user

import "errors"

// ErrUserNotFound is returned when the target user does not exist.
var ErrUserNotFound = errors.New("user not found")
