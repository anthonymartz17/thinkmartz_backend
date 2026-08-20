package auth

import "context"

// Authenticator defines methods a concrete service must implement to be used by Handler
type Authenticator interface {
	Register(ctx context.Context, email, password string) (*TokenPair, error)
}
