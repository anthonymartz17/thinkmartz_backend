package post_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post/mocks"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/go-chi/chi/v5"
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

func TestHandler_GetByID(t *testing.T) {
	t.Run("invalid post id", func(t *testing.T) {
		// arrange
		h, _ := newTestHandler(t)
		req := newGetByIDRequest("not-a-uuid")
		w := httptest.NewRecorder()

		// act
		h.GetByID(w, req)

		// assert
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("not found", func(t *testing.T) {
		// arrange
		h, mockSvc := newTestHandler(t)
		postID := uuid.New()
		mockSvc.EXPECT().GetByID(gomock.Any(), postID).Return(nil, post.ErrPostNotFound)
		req := newGetByIDRequest(postID.String())
		w := httptest.NewRecorder()

		// act
		h.GetByID(w, req)

		// assert
		assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	})

	t.Run("service error", func(t *testing.T) {
		// arrange
		h, mockSvc := newTestHandler(t)
		postID := uuid.New()
		mockSvc.EXPECT().GetByID(gomock.Any(), postID).Return(nil, assert.AnError)
		req := newGetByIDRequest(postID.String())
		w := httptest.NewRecorder()

		// act
		h.GetByID(w, req)

		// assert
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("success", func(t *testing.T) {
		// arrange
		h, mockSvc := newTestHandler(t)
		postID := uuid.New()
		wantPost := &post.Post{ID: postID, Content: "hello world"}
		mockSvc.EXPECT().GetByID(gomock.Any(), postID).Return(wantPost, nil)
		req := newGetByIDRequest(postID.String())
		w := httptest.NewRecorder()

		// act
		h.GetByID(w, req)

		// assert
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var got post.Post
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.Equal(t, wantPost.ID, got.ID)
		assert.Equal(t, wantPost.Content, got.Content)
	})
}

func newGetByIDRequest(postIDParam string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/posts/"+postIDParam, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("postID", postIDParam)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func newTestHandler(t *testing.T) (*post.Handler, *mocks.MockService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockService(ctrl)
	h := post.NewHandler(mockSvc, zap.NewNop())
	return h, mockSvc
}
