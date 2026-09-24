package post

import (
	"encoding/json"
	"net/http"

	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/middleware"
	"github.com/anthonymartz17/thinkmartz_backend/internal/transport/http/response"
	"go.uber.org/zap"
)

const (
	msgInvalidRequest = "invalid request"
	msgInternalServer = "internal server error"
)

// CreateInput is the decoded request body for Handler.Create.
type CreateInput struct {
	Content string `json:"content"`
}

// Handler is the post domain's HTTP layer.
type Handler struct {
	Service Service
	Logger  *zap.Logger
}

// NewHandler creates a new instance of Handler
func NewHandler(m Service, logger *zap.Logger) *Handler {
	return &Handler{
		Service: m,
		Logger:  logger,
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
		h.Logger.Error("failed to extract userID from context")
		return
	}

	post, err := h.Service.Create(ctx, userID, input.Content)

	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, msgInternalServer)
		h.Logger.Error("failed to create post", zap.Error(err))
		return
	}

	if err := response.WriteJSON(w, http.StatusCreated, post); err != nil {
		h.Logger.Error("failed to encode create post response", zap.Error(err))
	}
}
