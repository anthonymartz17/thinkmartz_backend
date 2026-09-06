package auth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/auth/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestHandler_Register(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		body := strings.NewReader(`{"email": "bad json`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		// act
		h.Register(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("validation fails", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		body := strings.NewReader(`{}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		// act
		h.Register(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var got auth.ValidationErrorResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.NotEmpty(t, got.Errors, "expected at least one field error")
	})

	t.Run("email already exists", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Register(gomock.Any(), gomock.Any()).
			Return(nil, auth.ErrEmailAlreadyExists)

		body := strings.NewReader(`{"email":"test@example.com","username":"testuser","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		// act
		h.Register(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("username already exists", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Register(gomock.Any(), gomock.Any()).
			Return(nil, auth.ErrUsernameAlreadyExists)

		body := strings.NewReader(`{"email":"test@example.com","username":"testuser","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		// act
		h.Register(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("service failure", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Register(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("something unexpected"))

		body := strings.NewReader(`{"email":"test@example.com","username":"testuser","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		// act
		h.Register(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Success", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Register(gomock.Any(), gomock.Any()).
			Return(&auth.Response{
				TokenPair: auth.TokenPair{
					AccessToken:  "FAKE.ACCESS.TOKEN",
					RefreshToken: "FAKE-REFRESH-TOKEN",
				},
			}, nil)

		body := strings.NewReader(`{"email":"test@example.com","username":"testuser","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		// act
		h.Register(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var got auth.RegisterResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.Equal(t, "FAKE.ACCESS.TOKEN", got.AccessToken)

		cookies := resp.Cookies()
		require.Len(t, cookies, 1, "expected exactly one cookie to be set")
		refreshCookie := cookies[0]
		assert.Equal(t, "refresh_token", refreshCookie.Name)
		assert.Equal(t, "FAKE-REFRESH-TOKEN", refreshCookie.Value)
		assert.True(t, refreshCookie.HttpOnly)
		assert.True(t, refreshCookie.Secure)
		assert.Equal(t, http.SameSiteStrictMode, refreshCookie.SameSite)
	})
}

func TestHandler_Login(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		body := strings.NewReader(`{"email": "bad json`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
		w := httptest.NewRecorder()

		// act
		h.Login(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("validation fails", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		body := strings.NewReader(`{}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
		w := httptest.NewRecorder()

		// act
		h.Login(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var got auth.ValidationErrorResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.NotEmpty(t, got.Errors, "expected at least one field error")
	})

	t.Run("user not found", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Login(gomock.Any(), gomock.Any()).
			Return(nil, auth.ErrUserNotFound)

		body := strings.NewReader(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
		w := httptest.NewRecorder()

		// act
		h.Login(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid password", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Login(gomock.Any(), gomock.Any()).
			Return(nil, auth.ErrInvalidPassword)

		body := strings.NewReader(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
		w := httptest.NewRecorder()

		// act
		h.Login(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("service failure", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Login(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("something unexpected"))

		body := strings.NewReader(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
		w := httptest.NewRecorder()

		// act
		h.Login(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Success", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		userID := uuid.New()
		createdAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		mockAuth.EXPECT().
			Login(gomock.Any(), gomock.Any()).
			Return(&auth.Response{
				User: auth.User{
					ID:        userID,
					Email:     "test@example.com",
					Username:  "testuser",
					CreatedAt: createdAt,
				},
				TokenPair: auth.TokenPair{
					AccessToken:  "FAKE.ACCESS.TOKEN",
					RefreshToken: "FAKE-REFRESH-TOKEN",
				},
			}, nil)

		body := strings.NewReader(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
		w := httptest.NewRecorder()

		// act
		h.Login(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var got auth.LoginResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.Equal(t, "FAKE.ACCESS.TOKEN", got.AccessToken)
		assert.Equal(t, userID, got.User.ID)
		assert.Equal(t, "test@example.com", got.User.Email)
		assert.Equal(t, "testuser", got.User.Username)
		assert.True(t, createdAt.Equal(got.User.CreatedAt))

		cookies := resp.Cookies()
		require.Len(t, cookies, 1, "expected exactly one cookie to be set")
		refreshCookie := cookies[0]
		assert.Equal(t, "refresh_token", refreshCookie.Name)
		assert.Equal(t, "FAKE-REFRESH-TOKEN", refreshCookie.Value)
		assert.True(t, refreshCookie.HttpOnly)
		assert.True(t, refreshCookie.Secure)
		assert.Equal(t, http.SameSiteStrictMode, refreshCookie.SameSite)
	})
}

func TestHandler_RefreshToken(t *testing.T) {
	t.Run("no cookie", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		w := httptest.NewRecorder()

		// act
		h.RefreshToken(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("refresh token not found", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			RefreshToken(gomock.Any(), "stale-opaque-token").
			Return(nil, auth.ErrRefreshTokenNotFound)

		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "stale-opaque-token"})
		w := httptest.NewRecorder()

		// act
		h.RefreshToken(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("service failure", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			RefreshToken(gomock.Any(), "some-opaque-token").
			Return(nil, errors.New("something unexpected"))

		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some-opaque-token"})
		w := httptest.NewRecorder()

		// act
		h.RefreshToken(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Success", func(t *testing.T) {
		// arrange
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			RefreshToken(gomock.Any(), "valid-opaque-token").
			Return(&auth.TokenPair{
				AccessToken:  "FAKE.ACCESS.TOKEN",
				RefreshToken: "FAKE-NEW-REFRESH-TOKEN",
			}, nil)

		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-opaque-token"})
		w := httptest.NewRecorder()

		// act
		h.RefreshToken(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var got auth.RefreshTokenResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.Equal(t, "FAKE.ACCESS.TOKEN", got.AccessToken)

		cookies := resp.Cookies()
		require.Len(t, cookies, 1, "expected exactly one cookie to be set")
		refreshCookie := cookies[0]
		assert.Equal(t, "refresh_token", refreshCookie.Name)
		assert.Equal(t, "FAKE-NEW-REFRESH-TOKEN", refreshCookie.Value)
		assert.True(t, refreshCookie.HttpOnly)
		assert.True(t, refreshCookie.Secure)
		assert.Equal(t, http.SameSiteStrictMode, refreshCookie.SameSite)
	})
}
