package auth

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest is the shape of data the client sends to POST /auth/register.
type RegisterRequest struct {
	Email    string
	Username string
	Password string
}

// User is Auth's minimal view of a user row — only what registration
// and login actually need. The fuller user profile (bio, follower
// count, etc.) belongs to the User domain's own model.
type User struct {
	ID           uuid.UUID
	Email        string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}
