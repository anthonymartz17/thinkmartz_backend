package auth

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

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

// Register hashes the password, intantiates a new user and saves it using UserRepository methods
func (s *Service) Register(ctx context.Context, email, password string) (*TokenPair, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return nil, fmt.Errorf("password hash: %w", err)
	}

	user := &User{
		Email:        email,
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

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
