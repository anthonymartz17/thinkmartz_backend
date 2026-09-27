package post

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	msgInvalidRequest = "invalid request"
	msgInternalServer = "internal server error"
	msgInvalidPostID  = "invalid post id"
	msgPostNotFound   = "post not found"
)

// CreateInput is the decoded request body for Handler.Create.
type CreateInput struct {
	Content string `json:"content"`
}

// Handler is the post domain's HTTP layer.
type Handler struct {
	Service Service
	logger  *zap.Logger
}

// NewHandler creates a new instance of Handler
func NewHandler(m Service, logger *zap.Logger) *Handler {
	return &Handler{
		Service: m,
		logger:  logger,
	}
}

// Create decodes a CreateInput, resolves the authenticated user from context, and creates a post.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {

	var input CreateInput
	ctx := r.Context()

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, msgInvalidRequest)
		return
	}

	if input.Content == "" {
		response.WriteError(w, http.StatusBadRequest, msgInvalidRequest)
		return

	}
	userID, ok := middleware.UserIDFromContext(ctx)

	if !ok {
		response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		h.logger.Error("failed to extract userID from context")
		return
	}

	post, err := h.Service.Create(ctx, userID, input.Content)

	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		h.logger.Error("failed to create post", zap.Error(err))
		return
	}

	if err := response.WriteJSON(w, http.StatusCreated, post); err != nil {
		h.logger.Error("failed to encode create post response", zap.Error(err))
	}
}

// GetByID resolves the post ID from the route and returns the matching post.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "postID"))

	if err != nil {
		response.WriteError(w, http.StatusBadRequest, msgInvalidPostID)
		return
	}

	post, err := h.Service.GetByID(r.Context(), postID)

	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.WriteError(w, http.StatusNotFound, msgPostNotFound)
			return
		}
		h.logger.Error("get post by id", zap.Stringer("post_id", postID), zap.Error(err))
		response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		return
	}

	if err := response.WriteJSON(w, http.StatusOK, post); err != nil {
		h.logger.Error("failed to encode GetByID response", zap.Error(err))
	}

}

// RegisterProtectedRoutes registers Handler's protected routes
func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/posts", h.Create)
	r.Get("/posts/{postID}", h.GetByID)
}
