package auth_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/auth/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister_PasswordTooLong(t *testing.T) {

	// arrange
	input := auth.RegisterInput{
		Email:    fmt.Sprintf("test@email.com%s", t.Name()),
		Username: fmt.Sprintf("fakeUsername%s", t.Name()),
		Password: strings.Repeat("a", 73),
	}
	ctx := t.Context()

	ctrl := gomock.NewController(t)
	mockRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)
	logger := zap.NewNop()

	newService := auth.NewService(mockRepo, mockTokenSrv, logger)

	// act
	_, gotErr := newService.Register(ctx, input)

	// assert
	assert.ErrorIs(t, gotErr, bcrypt.ErrPasswordTooLong)
}
func TestRegister(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		input := auth.RegisterInput{
			Email:    fmt.Sprintf("test@email.com%s", t.Name()),
			Username: fmt.Sprintf("fakeUsername%s", t.Name()),
			Password: strings.Repeat("a", 70),
		}
		var savedUserID uuid.UUID

		mockRepo.EXPECT().
			Save(ctx, gomock.Cond(func(x any) bool {
				u, ok := x.(*auth.User)
				return ok && u.Email == input.Email && bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password)) == nil
			})).
			DoAndReturn(func(_ context.Context, u *auth.User) error {
				savedUserID = uuid.New()
				u.ID = savedUserID
				return nil
			})

		mockTokenSrv.EXPECT().
			IssueAccessToken(gomock.Cond(func(x any) bool {
				id, ok := x.(uuid.UUID)
				return ok && id == savedUserID
			})).
			Return("FAKE.ACCESS.TOKEN", nil)

		mockTokenSrv.EXPECT().
			IssueRefreshToken(ctx, gomock.Cond(func(x any) bool {
				id, ok := x.(uuid.UUID)
				return ok && id == savedUserID
			})).
			Return("FAKE-REFRESH-TOKEN", nil)

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		// act
		gotAuthResp, err := svc.Register(ctx, input)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "FAKE.ACCESS.TOKEN", gotAuthResp.TokenPair.AccessToken)
		assert.Equal(t, "FAKE-REFRESH-TOKEN", gotAuthResp.TokenPair.RefreshToken)
	})

	t.Run("Fails when Repository Save returns an error", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)
		input := auth.RegisterInput{
			Email:    fmt.Sprintf("test@email.com%s", t.Name()),
			Username: fmt.Sprintf("fakeUsername%s", t.Name()),
			Password: strings.Repeat("a", 70),
		}

		mockRepo.EXPECT().
			Save(ctx, gomock.Any()).
			Return(errors.New("database failure"))

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		// act
		_, err := svc.Register(ctx, input)

		// assert
		assert.ErrorContains(t, err, "database failure")
	})

	t.Run("Triggers Delete Rollback when Refresh Token fails", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)
		input := auth.RegisterInput{
			Email:    fmt.Sprintf("test@email.com%s", t.Name()),
			Username: fmt.Sprintf("fakeUsername%s", t.Name()),
			Password: strings.Repeat("a", 70),
		}

		var savedUserID uuid.UUID

		mockRepo.EXPECT().
			Save(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, u *auth.User) error {
				savedUserID = uuid.New()
				u.ID = savedUserID
				return nil
			})

		mockTokenSrv.EXPECT().
			IssueAccessToken(gomock.Any()).
			Return("FAKE.ACCESS.TOKEN", nil)

		mockTokenSrv.EXPECT().
			IssueRefreshToken(ctx, gomock.Any()).
			Return("", errors.New("refresh token service unavailable"))

		mockRepo.EXPECT().
			Delete(ctx, gomock.Cond(func(x any) bool {
				id, ok := x.(uuid.UUID)
				return ok && id == savedUserID
			})).
			Return(nil)

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		// act
		_, err := svc.Register(ctx, input)

		// assert
		assert.ErrorContains(t, err, "refresh token: refresh token service unavailable")
	})
}

func TestLogin(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		email := fmt.Sprintf("test@email.com%s", t.Name())
		password := "correct-password"
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		userID := uuid.New()
		storedUser := &auth.User{
			ID:           userID,
			Email:        email,
			Username:     fmt.Sprintf("fakeUsername%s", t.Name()),
			PasswordHash: string(hash),
		}

		mockRepo.EXPECT().
			FindByEmail(ctx, email).
			Return(storedUser, nil)

		mockTokenSrv.EXPECT().
			IssueAccessToken(userID).
			Return("FAKE.ACCESS.TOKEN", nil)

		mockTokenSrv.EXPECT().
			IssueRefreshToken(ctx, userID).
			Return("FAKE-REFRESH-TOKEN", nil)

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		input := auth.LoginInput{Email: email, Password: password}

		// act
		gotResp, err := svc.Login(ctx, input)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, *storedUser, gotResp.User)
		assert.Equal(t, "FAKE.ACCESS.TOKEN", gotResp.TokenPair.AccessToken)
		assert.Equal(t, "FAKE-REFRESH-TOKEN", gotResp.TokenPair.RefreshToken)
	})

	t.Run("Fails when user is not found", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		email := fmt.Sprintf("test@email.com%s", t.Name())

		mockRepo.EXPECT().
			FindByEmail(ctx, email).
			Return(nil, auth.ErrUserNotFound)

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		input := auth.LoginInput{Email: email, Password: "whatever"}

		// act
		gotResp, err := svc.Login(ctx, input)

		// assert
		assert.Nil(t, gotResp)
		assert.ErrorIs(t, err, auth.ErrUserNotFound)
	})

	t.Run("Fails with invalid password", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		email := fmt.Sprintf("test@email.com%s", t.Name())
		hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
		require.NoError(t, err)

		storedUser := &auth.User{
			ID:           uuid.New(),
			Email:        email,
			Username:     fmt.Sprintf("fakeUsername%s", t.Name()),
			PasswordHash: string(hash),
		}

		mockRepo.EXPECT().
			FindByEmail(ctx, email).
			Return(storedUser, nil)

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		input := auth.LoginInput{Email: email, Password: "wrong-password"}

		// act
		gotResp, err := svc.Login(ctx, input)

		// assert
		assert.Nil(t, gotResp)
		assert.ErrorIs(t, err, auth.ErrInvalidPassword)
	})

	t.Run("Fails when IssueAccessToken errors", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		email := fmt.Sprintf("test@email.com%s", t.Name())
		password := "correct-password"
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		userID := uuid.New()
		storedUser := &auth.User{
			ID:           userID,
			Email:        email,
			Username:     fmt.Sprintf("fakeUsername%s", t.Name()),
			PasswordHash: string(hash),
		}

		mockRepo.EXPECT().
			FindByEmail(ctx, email).
			Return(storedUser, nil)

		mockTokenSrv.EXPECT().
			IssueAccessToken(userID).
			Return("", errors.New("access token service unavailable"))

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		input := auth.LoginInput{Email: email, Password: password}

		// act
		gotResp, err := svc.Login(ctx, input)

		// assert
		assert.Nil(t, gotResp)
		assert.ErrorContains(t, err, "access token service unavailable")
	})

	t.Run("Fails when IssueRefreshToken errors", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		email := fmt.Sprintf("test@email.com%s", t.Name())
		password := "correct-password"
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		userID := uuid.New()
		storedUser := &auth.User{
			ID:           userID,
			Email:        email,
			Username:     fmt.Sprintf("fakeUsername%s", t.Name()),
			PasswordHash: string(hash),
		}

		mockRepo.EXPECT().
			FindByEmail(ctx, email).
			Return(storedUser, nil)

		mockTokenSrv.EXPECT().
			IssueAccessToken(userID).
			Return("FAKE.ACCESS.TOKEN", nil)

		mockTokenSrv.EXPECT().
			IssueRefreshToken(ctx, userID).
			Return("", errors.New("refresh token service unavailable"))

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		input := auth.LoginInput{Email: email, Password: password}

		// act
		gotResp, err := svc.Login(ctx, input)

		// assert
		assert.Nil(t, gotResp)
		assert.ErrorContains(t, err, "refresh token service unavailable")
	})
}

func TestRefreshToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		userID := uuid.New()

		mockTokenSrv.EXPECT().
			ValidateRefreshToken(ctx, "old-opaque-token").
			Return(userID, nil)

		mockTokenSrv.EXPECT().
			IssueAccessToken(userID).
			Return("FAKE.ACCESS.TOKEN", nil)

		mockTokenSrv.EXPECT().
			IssueRefreshToken(ctx, userID).
			Return("FAKE-NEW-REFRESH-TOKEN", nil)

		mockTokenSrv.EXPECT().
			InvalidateRefreshToken(ctx, "old-opaque-token").
			Return(nil)

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		// act
		gotTokenPair, gotErr := svc.RefreshToken(ctx, "old-opaque-token")

		// assert
		assert.NoError(t, gotErr)
		assert.Equal(t, "FAKE.ACCESS.TOKEN", gotTokenPair.AccessToken)
		assert.Equal(t, "FAKE-NEW-REFRESH-TOKEN", gotTokenPair.RefreshToken)
	})

	t.Run("Fails when refresh token is invalid", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		mockTokenSrv.EXPECT().
			ValidateRefreshToken(ctx, "bad-opaque-token").
			Return(uuid.Nil, auth.ErrRefreshTokenNotFound)

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		// act
		gotTokenPair, gotErr := svc.RefreshToken(ctx, "bad-opaque-token")

		// assert
		assert.Nil(t, gotTokenPair)
		assert.ErrorIs(t, gotErr, auth.ErrRefreshTokenNotFound)
	})

	t.Run("Fails when IssueAccessToken errors", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		userID := uuid.New()

		mockTokenSrv.EXPECT().
			ValidateRefreshToken(ctx, "old-opaque-token").
			Return(userID, nil)

		mockTokenSrv.EXPECT().
			IssueAccessToken(userID).
			Return("", errors.New("access token service unavailable"))

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		// act
		gotTokenPair, gotErr := svc.RefreshToken(ctx, "old-opaque-token")

		// assert
		assert.Nil(t, gotTokenPair)
		assert.ErrorContains(t, gotErr, "access token service unavailable")
	})

	t.Run("Fails when IssueRefreshToken errors", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		userID := uuid.New()

		mockTokenSrv.EXPECT().
			ValidateRefreshToken(ctx, "old-opaque-token").
			Return(userID, nil)

		mockTokenSrv.EXPECT().
			IssueAccessToken(userID).
			Return("FAKE.ACCESS.TOKEN", nil)

		mockTokenSrv.EXPECT().
			IssueRefreshToken(ctx, userID).
			Return("", errors.New("refresh token service unavailable"))

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		// act
		gotTokenPair, gotErr := svc.RefreshToken(ctx, "old-opaque-token")

		// assert
		assert.Nil(t, gotTokenPair)
		assert.ErrorContains(t, gotErr, "refresh token service unavailable")
	})

	t.Run("Succeeds even when InvalidateRefreshToken fails", func(t *testing.T) {
		// arrange
		ctx := t.Context()
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockUserRepository(ctrl)
		mockTokenSrv := mocks.NewMockTokenIssuer(ctrl)

		userID := uuid.New()

		mockTokenSrv.EXPECT().
			ValidateRefreshToken(ctx, "old-opaque-token").
			Return(userID, nil)

		mockTokenSrv.EXPECT().
			IssueAccessToken(userID).
			Return("FAKE.ACCESS.TOKEN", nil)

		mockTokenSrv.EXPECT().
			IssueRefreshToken(ctx, userID).
			Return("FAKE-NEW-REFRESH-TOKEN", nil)

		mockTokenSrv.EXPECT().
			InvalidateRefreshToken(ctx, "old-opaque-token").
			Return(errors.New("redis unavailable"))

		svc := auth.NewService(mockRepo, mockTokenSrv, zap.NewNop())

		// act
		gotTokenPair, gotErr := svc.RefreshToken(ctx, "old-opaque-token")

		// assert
		assert.NoError(t, gotErr, "invalidation failure should only be logged, not returned")
		assert.Equal(t, "FAKE.ACCESS.TOKEN", gotTokenPair.AccessToken)
		assert.Equal(t, "FAKE-NEW-REFRESH-TOKEN", gotTokenPair.RefreshToken)
	})
}
