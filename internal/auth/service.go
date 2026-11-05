package auth

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const _bcryptCost = bcrypt.DefaultCost

// validates Service implements Authenticator
var _ Authenticator = (*Service)(nil)

// Service handles Auth business logic — user registration, login,
// and JWT issuing — by orchestrating calls to a Repository.
type Service struct {
	repo     Repository
	token    TokenIssuer
	sessions SessionRepository
	logger   *zap.Logger
	// dummyHash is a constant-cost stand-in compared against on a not-found
	// user, so login timing doesn't reveal whether the email exists.
	dummyHash []byte
}

// NewService creates a new Service
func NewService(r Repository, t TokenIssuer, sessions SessionRepository, l *zap.Logger) (*Service, error) {

	const dummyPassword = "timing-attack-guard"
	dummyHash, err := bcrypt.GenerateFromPassword([]byte(dummyPassword), _bcryptCost)

	if err != nil {
		return nil, fmt.Errorf("generate dummy hash: %w", err)
	}

	return &Service{
		repo:      r,
		token:     t,
		sessions:  sessions,
		logger:    l,
		dummyHash: dummyHash,
	}, nil
}

// TokenPair contains both string tokens. avoids having to return two tokens of the same type that can be easily
// confused in order for Register method
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// RegisterInput is the data Service.Register needs to create a new user,
// decoupled from the transport layer's request shape.
type RegisterInput struct {
	Email    string
	Username string
	Password string
}

// LoginInput is the data Service.Login needs to sign in a user,
// decoupled from the transport layer's request shape.
type LoginInput struct {
	Email    string
	Password string
}

// Response bundles the user record with the issued token pair,
// returned by both Register and Login.
type Response struct {
	User      User
	TokenPair TokenPair
}

// Register hashes the password, intantiates a new user and saves it using Repository methods
func (s *Service) Register(ctx context.Context, input RegisterInput) (*Response, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), _bcryptCost)

	if err != nil {
		return nil, fmt.Errorf("password hash: %w", err)
	}

	user := &User{
		Email:        input.Email,
		Username:     input.Username,
		PasswordHash: string(hash),
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	accessToken, err := s.token.IssueAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	rollback := func(cause error) error {
		if delErr := s.repo.Delete(ctx, user.ID); delErr != nil {
			s.logger.Error(
				"user creation rollback",
				zap.Error(delErr),
				zap.NamedError("refresh_token_issue", cause),
				zap.Stringer("userId", user.ID),
			)
		}
		return fmt.Errorf("refresh token: %w", cause)
	}

	refreshToken, err := s.token.GenerateRefreshToken()
	if err != nil {
		return nil, rollback(err)
	}

	if err := s.sessions.Set(ctx, refreshToken, user.ID); err != nil {
		return nil, rollback(err)
	}

	return &Response{
		User:      *user,
		TokenPair: TokenPair{AccessToken: accessToken, RefreshToken: refreshToken},
	}, nil
}

// Login finds the user by email, validates the password, and returns a Response on success.
func (s *Service) Login(ctx context.Context, input LoginInput) (*Response, error) {

	user, err := s.repo.FindByEmail(ctx, input.Email)

	if err != nil {

		if errors.Is(err, ErrUserNotFound) {
			// timing attack guard
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(input.Password))
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrInvalidPassword
		}

		return nil, fmt.Errorf("compare password hash: %w", err)
	}

	accessToken, err := s.token.IssueAccessToken(user.ID)

	if err != nil {
		return nil, err
	}

	refreshToken, err := s.token.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if err := s.sessions.Set(ctx, refreshToken, user.ID); err != nil {
		return nil, err
	}

	return &Response{
		User: *user,
		TokenPair: TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	}, nil

}

// RefreshToken atomically validates and invalidates the received refresh
// token (so it can't be redeemed twice), issues a new access and refresh
// token pair, and returns it on success.
func (s *Service) RefreshToken(ctx context.Context, opaque string) (*TokenPair, error) {

	userID, err := s.sessions.GetDel(ctx, opaque)

	if err != nil {
		return nil, err
	}

	accessToken, err := s.token.IssueAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.token.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if err := s.sessions.Set(ctx, refreshToken, userID); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Logout ends the session associated with the given refresh token.
// It is idempotent: invalidating an already-expired or unknown token is not an error.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Clear(ctx, token)

}
