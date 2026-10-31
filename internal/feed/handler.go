package feed

import (
	"net/http"

	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	msgUnauthorized   = "unauthorized"
	msgInternalServer = "internal server error"
	msgInvalidCursor  = "invalid cursor"
)

// feedResponse is the GET /feed response body.
type feedResponse struct {
	Data       []post.Post `json:"data"`
	NextCursor *string     `json:"next_cursor"`
}

// Handler is the feed domain's HTTP layer.
type Handler struct {
	service Reader
	logger  *zap.Logger
}

// NewHandler creates a new instance of Handler.
func NewHandler(s Reader, l *zap.Logger) *Handler {
	return &Handler{
		service: s,
		logger:  l,
	}
}

// GetFeed handles GET /feed: one page of the authenticated user's home
// timeline, newest first. A missing cursor requests the first page.
func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.UserIDFromContext(ctx)

	if !ok {
		response.WriteError(w, http.StatusUnauthorized, msgUnauthorized)
		h.logger.Error("failed to extract userID from context")
		return
	}

	var cursor *Cursor
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		c, err := DecodeCursor(raw)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, msgInvalidCursor)
			return
		}
		cursor = &c
	}

	posts, next, err := h.service.GetFeed(r.Context(), userID, cursor)
	if err != nil {
		h.logger.Error("get feed", zap.Error(err), zap.Stringer("userID", userID))
		response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		return
	}

	if posts == nil {
		posts = []post.Post{}
	}

	body := feedResponse{Data: posts}
	if next != nil {
		encoded := next.Encode()
		body.NextCursor = &encoded
	}

	if err := response.WriteJSON(w, http.StatusOK, body); err != nil {
		h.logger.Error("failed to encode get feed response", zap.Error(err))
	}
}

// RegisterProtectedRoutes registers Handler's protected routes.
func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/feed", h.GetFeed)
}
