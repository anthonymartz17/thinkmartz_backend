// Package auth handles user registration, login, JWT issuing/validation,
// and session management.
package auth

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the methods a concrete repository must implement to be used by Service.
type Repository interface {
	Save(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}
