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

		gotAuthResp, err := svc.Register(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, "FAKE.ACCESS.TOKEN", gotAuthResp.TokenPair.AccessToken)
		assert.Equal(t, "FAKE-REFRESH-TOKEN", gotAuthResp.TokenPair.RefreshToken)
	})

	t.Run("Fails when Repository Save returns an error", func(t *testing.T) {
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

		_, err := svc.Register(ctx, input)

		assert.ErrorContains(t, err, "database failure")
	})

	t.Run("Triggers Delete Rollback when Refresh Token fails", func(t *testing.T) {
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

		_, err := svc.Register(ctx, input)

		assert.ErrorContains(t, err, "refresh token: refresh token service unavailable")
	})
}
