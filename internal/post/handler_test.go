package post_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post/mocks"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestCreate(t *testing.T) {
	t.Run("malformed request", func(t *testing.T) {
		// arrange
		h, _ := newTestHandler(t)

		body := strings.NewReader(`{"content":`)
		req := httptest.NewRequest(http.MethodPost, "/post", body)
		w := httptest.NewRecorder()

		// act
		h.Create(w, req)

		// assert
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("content is empty", func(t *testing.T) {
		// arrange
		h, _ := newTestHandler(t)

		body := strings.NewReader(`{"content":""}`)
		req := httptest.NewRequest(http.MethodPost, "/post", body)
		w := httptest.NewRecorder()

		// act
		h.Create(w, req)

		// assert
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("unable to extract user id from context", func(t *testing.T) {
		// arrange
		h, _ := newTestHandler(t)

		body := strings.NewReader(`{"content":"hello world"}`)
		req := httptest.NewRequest(http.MethodPost, "/post", body)
		w := httptest.NewRecorder()

		// act
		h.Create(w, req)

		// assert
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("h.Service.Create fails", func(t *testing.T) {
		// arrange
		h, mockSvc := newTestHandler(t)

		userID := uuid.New()
		mockSvc.EXPECT().Create(gomock.Any(), userID, "hello world").Return(nil, assert.AnError)

		body := strings.NewReader(`{"content":"hello world"}`)
		req := httptest.NewRequest(http.MethodPost, "/post", body)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))
		w := httptest.NewRecorder()

		// act
		h.Create(w, req)

		// assert
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("success", func(t *testing.T) {
		// arrange
		h, mockSvc := newTestHandler(t)

		userID := uuid.New()
		wantPost := &post.Post{
			ID:      uuid.New(),
			UserID:  userID,
			Content: "hello world",
		}
		mockSvc.EXPECT().Create(gomock.Any(), userID, "hello world").Return(wantPost, nil)

		body := strings.NewReader(`{"content":"hello world"}`)
		req := httptest.NewRequest(http.MethodPost, "/post", body)
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))
		w := httptest.NewRecorder()

		// act
		h.Create(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var got post.Post
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.Equal(t, wantPost.ID, got.ID)
		assert.Equal(t, wantPost.UserID, got.UserID)
		assert.Equal(t, wantPost.Content, got.Content)
	})
}

func newTestHandler(t *testing.T) (*post.Handler, *mocks.MockService) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockService(ctrl)
	h := post.NewHandler(mockSvc, zap.NewNop())
	return h, mockSvc
}
