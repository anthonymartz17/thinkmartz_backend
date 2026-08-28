package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// RegisterResponse represents a valid  registration response
type RegisterResponse struct {
	AccessToken string `json:"access_token"`
}

// FieldError represents a single validation failure on one field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResponse is the JSON body returned when request
// validation fails.
type ValidationErrorResponse struct {
	Errors []FieldError `json:"errors"`
}

// RegisterRequest is the shape of data the client sends to POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=30"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

// Handler translates HTTP requests into Authenticator calls and writes the
// resulting response
type Handler struct {
	service Authenticator
	logger  *zap.Logger
}

// NewHandler creates and returns a new Handler
func NewHandler(srv Authenticator, l *zap.Logger) *Handler {
	return &Handler{
		service: srv,
		logger:  l,
	}
}

// Register handles registration request-response requests
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {

	// 	w.WriteHeader(http.StatusCreated)
	// w.Header().Set("Content-Type", "application/json")

	// if err := json.NewEncoder(w).Encode(&RegisterResponse{AccessToken: "tokenPair.AccessToken example"}); err != nil { //nolint:gosec // access token is intentionally returned to the client in the response body
	// 	h.logger.Error("failed to encode register response", zap.Error(err))
	// }

	// return

	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := Validate.Struct(req); err != nil {
		var validationErrs validator.ValidationErrors

		if errors.As(err, &validationErrs) {
			resp := ValidationErrorResponse{}
			for _, fe := range validationErrs {
				resp.Errors = append(resp.Errors, FieldError{
					Field:   fe.Field(),
					Message: fmt.Sprintf("failed on the '%s' rule", fe.Tag()),
				})
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			if err := json.NewEncoder(w).Encode(resp); err != nil {
				h.logger.Error("failed to encode validation error response", zap.Error(err))
			}
			return
		}

		http.Error(w, "invalid request", http.StatusBadRequest)
		return

	}

	input := &RegisterInput{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	}

	tokenPair, err := h.service.Register(r.Context(), *input)

	if err != nil {
		switch {
		case errors.Is(err, ErrEmailAlreadyExists):
			http.Error(w, "email already exists", http.StatusConflict)
		case errors.Is(err, ErrUsernameAlreadyExists):
			http.Error(w, "username already exists", http.StatusConflict)
		default:
			h.logger.Error("register failed", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		HttpOnly: true,
		Secure:   true,
		Value:    tokenPair.RefreshToken,
		Path:     "/auth/refresh",
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(&RegisterResponse{AccessToken: tokenPair.AccessToken}); err != nil { //nolint:gosec // access token is intentionally returned to the client in the response body
		h.logger.Error("failed to encode register response", zap.Error(err))
	}
}

// RegisterPublicRoutes registers Handler's public routes
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/auth/register", h.Register)
}

// RegisterProtectedRoutes registers Handler's protected routes
func (h *Handler) RegisterProtectedRoutes(_ chi.Router) {
	// to be implemented
}

// RegisterRefreshRoutes registers refresh cookie route which is particular to auth Handler
func (h *Handler) RegisterRefreshRoutes(_ chi.Router) {
	// to be implemented
}
