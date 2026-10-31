package feed_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/anthonymartz17/thinkmartz_backend/internal/feed"
	"github.com/anthonymartz17/thinkmartz_backend/internal/feed/mocks"
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
)

// feedBody mirrors the GET /feed response for assertions.
type feedBody struct {
	Data       []json.RawMessage `json:"data"`
	NextCursor *string           `json:"next_cursor"`
}

func newTestHandler(t *testing.T) (*feed.Handler, *mocks.MockReader) {
	t.Helper()
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockReader(ctrl)
	return feed.NewHandler(reader, zap.NewNop()), reader
}

// newFeedRequest builds a GET request with userID in the context, the way the
// auth middleware would.
func newFeedRequest(target string, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	return req.WithContext(middleware.ContextWithUserID(req.Context(), userID))
}

func decodeFeedBody(t *testing.T, rec *httptest.ResponseRecorder) feedBody {
	t.Helper()
	var body feedBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestHandler_GetFeed_MissingUser(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	rec := httptest.NewRecorder()

	h.GetFeed(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.JSONEq(t, `{"error":"unauthorized"}`, rec.Body.String())
}

func TestHandler_GetFeed_InvalidCursor(t *testing.T) {
	h, _ := newTestHandler(t)
	rec := httptest.NewRecorder()

	h.GetFeed(rec, newFeedRequest("/feed?cursor=!!!", uuid.New()))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"invalid cursor"}`, rec.Body.String())
}

func TestHandler_GetFeed_ServiceError(t *testing.T) {
	h, reader := newTestHandler(t)
	userID := uuid.New()
	reader.EXPECT().
		GetFeed(gomock.Any(), userID, nil).
		Return(nil, nil, errors.New("boom"))
	rec := httptest.NewRecorder()

	h.GetFeed(rec, newFeedRequest("/feed", userID))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_GetFeed_EmptyFeed(t *testing.T) {
	h, reader := newTestHandler(t)
	userID := uuid.New()
	reader.EXPECT().
		GetFeed(gomock.Any(), userID, nil).
		Return(nil, nil, nil)
	rec := httptest.NewRecorder()

	h.GetFeed(rec, newFeedRequest("/feed", userID))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":[],"next_cursor":null}`, rec.Body.String())
}

func TestHandler_GetFeed_LastPage(t *testing.T) {
	h, reader := newTestHandler(t)
	userID := uuid.New()
	posts := []post.Post{{ID: uuid.New()}, {ID: uuid.New()}}
	reader.EXPECT().
		GetFeed(gomock.Any(), userID, nil).
		Return(posts, nil, nil)
	rec := httptest.NewRecorder()

	h.GetFeed(rec, newFeedRequest("/feed", userID))

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeFeedBody(t, rec)
	assert.Len(t, body.Data, len(posts))
	assert.Nil(t, body.NextCursor)
}

func TestHandler_GetFeed_MorePages(t *testing.T) {
	h, reader := newTestHandler(t)
	userID := uuid.New()
	posts := []post.Post{{ID: uuid.New()}}
	next := &feed.Cursor{
		CreatedAt: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		PostID:    uuid.New(),
	}
	reader.EXPECT().
		GetFeed(gomock.Any(), userID, nil).
		Return(posts, next, nil)
	rec := httptest.NewRecorder()

	h.GetFeed(rec, newFeedRequest("/feed", userID))

	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeFeedBody(t, rec)
	require.NotNil(t, body.NextCursor)
	assert.Equal(t, next.Encode(), *body.NextCursor)
}

func TestHandler_GetFeed_ValidCursorIsPassedToService(t *testing.T) {
	h, reader := newTestHandler(t)
	userID := uuid.New()
	cursor := feed.Cursor{
		CreatedAt: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		PostID:    uuid.New(),
	}
	reader.EXPECT().
		GetFeed(gomock.Any(), userID, &cursor).
		Return(nil, nil, nil)
	rec := httptest.NewRecorder()

	h.GetFeed(rec, newFeedRequest("/feed?cursor="+cursor.Encode(), userID))

	assert.Equal(t, http.StatusOK, rec.Code)
}
