package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)
var (
	ErrAlreadyExists   = errors.New("already exists")

)


type Repository struct{
	Pool *pgxpool.Pool
}


func NewRepository(p *pgxpool.Pool) *Repository{
	return &Repository{
     Pool:p,
	}
}

func(r *Repository)Save(ctx context.Context, user *User) error{

	query:= `
		INSERT INTO users (email, username, password_hash)
		VALUES ($1,$2,$3)
		RETURNING id,created_at
	`
	err:= r.Pool.QueryRow(
		ctx,
		query,
		user.Email,
		user.Username,
		user.PasswordHash,
	).Scan(&user.ID,&user.CreatedAt)

var pgErr *pgconn.PgError

if err!= nil {

  if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAlreadyExists
	}

	return fmt.Errorf("save user: %w", err)
}

	return nil
}

