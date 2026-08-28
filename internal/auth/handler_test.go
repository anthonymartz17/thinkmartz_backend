package auth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/auth"
	"github.com/anthonymartz17/thinkmartz_backend/internal/auth/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestHandler_Register(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		body := strings.NewReader(`{"email": "bad json`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		h.Register(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("validation fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		body := strings.NewReader(`{}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		h.Register(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var got auth.ValidationErrorResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.NotEmpty(t, got.Errors, "expected at least one field error")
	})

	t.Run("email already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Register(gomock.Any(), gomock.Any()).
			Return(nil, auth.ErrEmailAlreadyExists)

		body := strings.NewReader(`{"email":"test@example.com","username":"testuser","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		h.Register(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("username already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Register(gomock.Any(), gomock.Any()).
			Return(nil, auth.ErrUsernameAlreadyExists)

		body := strings.NewReader(`{"email":"test@example.com","username":"testuser","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		h.Register(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("service failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockAuth := mocks.NewMockAuthenticator(ctrl)

		h := auth.NewHandler(mockAuth, zap.NewNop())

		mockAuth.EXPECT().
			Register(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("something unexpected"))

		body := strings.NewReader(`{"email":"test@example.com","username":"testuser","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
		w := httptest.NewRecorder()

		h.Register(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Success", func(t *testing.T) {
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

		h.Register(w, req)

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
