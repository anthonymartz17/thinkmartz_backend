package auth

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidPassword indicates the provided password did not match the stored hash.
	ErrInvalidPassword = errors.New("password is invalid")
)

// validates Service implements Authenticator
var _ Authenticator = (*Service)(nil)

// Service handles Auth business logic — user registration, login,
// and JWT issuing — by orchestrating calls to a UserRepository.
type Service struct {
	repo   UserRepository
	token  TokenIssuer
	logger *zap.Logger
}

// NewService creates a new Service
func NewService(r UserRepository, t TokenIssuer, l *zap.Logger) *Service {
	return &Service{
		repo:   r,
		token:  t,
		logger: l,
	}
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

// LoginInput is the data Service.Login needs to create a new user,
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

// Register hashes the password, intantiates a new user and saves it using UserRepository methods
func (s *Service) Register(ctx context.Context, input RegisterInput) (*Response, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

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

	refreshToken, err := s.token.IssueRefreshToken(ctx, user.ID)
	if err != nil {

		if delErr := s.repo.Delete(ctx, user.ID); delErr != nil {
			s.logger.Error(
				"user creation rollback",
				zap.Error(delErr),
				zap.NamedError("refresh_token_issue", err),
				zap.Stringer("userId", user.ID),
			)
		}

		return nil, fmt.Errorf("refresh token: %w", err)
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
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	accessToken, err := s.token.IssueAccessToken(user.ID)

	if err != nil {
		return nil, err
	}

	refreshToken, err := s.token.IssueRefreshToken(ctx, user.ID)
	if err != nil {
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

// RefreshToken validates received refresh token
// issues both access and refresh token
// invalidates old token
// returns a TokenPair on success and error on failure.
func (s *Service) RefreshToken(ctx context.Context, opaque string) (*TokenPair, error) {

	userID, err := s.token.ValidateRefreshToken(ctx, opaque)

	if err != nil {
		return nil, err
	}

	accessToken, err := s.token.IssueAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.token.IssueRefreshToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.token.InvalidateRefreshToken(ctx, opaque); err != nil {
		s.logger.Warn(
			"failed to invalidate old refresh token after rotation",
			zap.Error(err),
			zap.String("stale_opaque_prefix", opaque[:8]),
		)
	}
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Logout ends the session associated with the given refresh token.
// It is idempotent: invalidating an already-expired or unknown token is not an error.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.token.InvalidateRefreshToken(ctx, token)

}
