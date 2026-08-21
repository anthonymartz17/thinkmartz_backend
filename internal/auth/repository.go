// Package auth handles user registration, login, JWT issuing/validation,
// and session management.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrAlreadyExists is returned when a user's email or username is
	// already taken.
	ErrAlreadyExists = errors.New("already exists")
	// ErrEmailAlreadyExists is returned when a user's email is already taken.
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrUsernameAlreadyExists is returned when a username is already taken.
	ErrUsernameAlreadyExists = errors.New("username already exists")

	// ErrUserNotFound is returned when no user matches the given email.
	ErrUserNotFound = errors.New("user not found")
)

// Checks if  Repository implements UserRepository
var _ UserRepository = (*Repository)(nil)

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
			switch pgErr.ConstraintName {
			case "users_email_key":
				return ErrEmailAlreadyExists
			case "users_username_key":
				return ErrUsernameAlreadyExists
			default:
				return ErrAlreadyExists
			}
		}

		return fmt.Errorf("save user: %w", err)
	}

	return nil
}

// FindByEmail returns a full user by email if exists
func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {

	var user User
	query := `
	SELECT id, email, username, password_hash, created_at FROM users 
	WHERE email = $1
	`

	err := r.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("find by email: %w", err)

	}

	return &user, nil

}

// Delete removes a user from users table
func (r *Repository) Delete(ctx context.Context, userID uuid.UUID) error {

	query := `
	DELETE FROM users
	WHERE id = $1
	`
	rowsDeleted, err := r.Pool.Exec(ctx, query, userID)

	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if rowsDeleted.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}
