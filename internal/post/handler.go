package post

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/response"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/validation"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	msgInvalidRequest = "invalid request"
	msgInternalServer = "internal server error"
	msgInvalidPostID  = "invalid post id"
	msgPostNotFound   = "post not found"
	msgForbidden      = "you can only edit your own posts"
)

// CreateInput is the decoded request body for Handler.Create.
type CreateInput struct {
	Content string `json:"content" validate:"required,max=280"`
}

// updateRequest is the decoded request body for Handler.Update.

type updateRequest struct {
	Content string `json:"content" validate:"required,max=280"`
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
	input.Content = strings.TrimSpace(input.Content)
	if err := validation.Validate.Struct(input); err != nil {
		var validationErrs validator.ValidationErrors

		if errors.As(err, &validationErrs) {
			if err := response.WriteJSON(w, http.StatusBadRequest, validation.ToErrorResponse(validationErrs)); err != nil {
				h.logger.Error("failed to encode validation error response", zap.Error(err))
			}
			return
		}

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

// Update replaces the content of a post owned by the authenticated user.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {

	var req updateRequest
	ctx := r.Context()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, msgInvalidRequest)
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if err := validation.Validate.Struct(req); err != nil {
		var validationErrs validator.ValidationErrors

		if errors.As(err, &validationErrs) {
			if err := response.WriteJSON(w, http.StatusBadRequest, validation.ToErrorResponse(validationErrs)); err != nil {
				h.logger.Error("failed to encode validation error response", zap.Error(err))
			}
			return
		}

		response.WriteError(w, http.StatusBadRequest, msgInvalidRequest)
		return
	}

	userID, ok := middleware.UserIDFromContext(ctx)

	if !ok {
		response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		h.logger.Error("failed to extract userID from context")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(r, "postID"))

	if err != nil {
		response.WriteError(w, http.StatusBadRequest, msgInvalidPostID)
		return
	}

	input := UpdateInput{
		PostID:  postID,
		UserID:  userID,
		Content: req.Content,
	}

	post, err := h.Service.Update(ctx, input)

	if err != nil {

		if errors.Is(err, ErrForbidden) {
			response.WriteError(w, http.StatusForbidden, msgForbidden)
			return
		}

		if errors.Is(err, ErrPostNotFound) {
			response.WriteError(w, http.StatusNotFound, msgPostNotFound)
			return
		}

		response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		h.logger.Error("failed to update post", zap.String("post_id", postID.String()), zap.Error(err))
		return
	}

	if err := response.WriteJSON(w, http.StatusOK, post); err != nil {
		h.logger.Error("failed to encode update post response", zap.Error(err))
	}
}

// Delete handles DELETE /posts/{postID}. Owner only. Returns 204 on success.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, msgInvalidPostID)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		h.logger.Error("user ID missing from context")
		response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		return
	}

	input := DeleteInput{
		PostID: postID,
		UserID: userID,
	}
	err = h.Service.Delete(r.Context(), input)

	if err != nil {
		switch {
		case errors.Is(err, ErrPostNotFound):
			response.WriteError(w, http.StatusNotFound, msgPostNotFound)
		case errors.Is(err, ErrForbidden):
			response.WriteError(w, http.StatusForbidden, msgForbidden)
		default:
			h.logger.Error("delete post",
				zap.String("post_id", postID.String()),
				zap.Error(err),
			)
			response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterProtectedRoutes registers Handler's protected routes
func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/posts", h.Create)
	r.Get("/posts/{postID}", h.GetByID)
	r.Patch("/posts/{postID}", h.Update)
	r.Delete("/posts/{postID}", h.Delete)

}
