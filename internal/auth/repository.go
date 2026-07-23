package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrAlreadyExists is returned when a user's email or username is
	// already taken.
	ErrAlreadyExists = errors.New("already exists")
)

// Repository provides a connection pool and methods to interact with database
type Repository struct {
	Pool *pgxpool.Pool
}

// NewRepository creates and returns a new Repository
func NewRepository(p *pgxpool.Pool) *Repository {
	return &Repository{
		Pool: p,
	}
}

// Save stores a new user to database
func (r *Repository) Save(ctx context.Context, user *User) error {

	query := `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1,$2,$3)
		RETURNING id,created_at
	`
	err := r.Pool.QueryRow(
		ctx,
		query,
		user.Email,
		user.Username,
		user.PasswordHash,
	).Scan(&user.ID, &user.CreatedAt)

	var pgErr *pgconn.PgError

	if err != nil {

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}

		return fmt.Errorf("save user: %w", err)
	}

	return nil
}
