package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	msgInvalidBody        = "invalid request body"
	msgInvalidRequest     = "invalid request"
	msgInvalidSession     = "invalid session"
	msgEmailExists        = "email already exists"
	msgUsernameExists     = "username already exists"
	msgInvalidCredentials = "email or password is invalid"
	msgInternalServer     = "internal server error"
)

// UserResponse is the client-safe projection of a User returned in auth responses.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterResponse represents a valid registration response.
type RegisterResponse struct {
	AccessToken string       `json:"access_token"`
	User        UserResponse `json:"user"`
}

// LoginResponse represents a valid login response.
type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	User        UserResponse `json:"user"`
}

// RefreshTokenResponse represents a valid refresh token response.
type RefreshTokenResponse struct {
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

// LoginRequest is the shape of data the client sends to POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
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

	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidBody)
		return
	}

	if err := Validate.Struct(req); err != nil {
		var validationErrs validator.ValidationErrors

		if errors.As(err, &validationErrs) {
			if err := writeJSON(w, http.StatusBadRequest, toValidationErrorResponse(validationErrs)); err != nil {
				h.logger.Error("failed to encode validation error response", zap.Error(err))
			}
			return
		}

		writeError(w, http.StatusBadRequest, msgInvalidRequest)
		return

	}

	authResp, err := h.service.Register(r.Context(), toRegisterInput(req))

	if err != nil {
		switch {
		case errors.Is(err, ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, msgEmailExists)
		case errors.Is(err, ErrUsernameAlreadyExists):
			writeError(w, http.StatusConflict, msgUsernameExists)
		default:
			h.logger.Error("register failed", zap.Error(err))
			writeError(w, http.StatusInternalServerError, msgInternalServer)
		}
		return
	}

	setRefreshCookie(w, authResp.TokenPair.RefreshToken)

	registerResp := &RegisterResponse{
		AccessToken: authResp.TokenPair.AccessToken,
		User:        toUserResponse(authResp.User),
	}
	if err := writeJSON(w, http.StatusCreated, registerResp); err != nil { //nolint:gosec // access token is intentionally returned to the client in the response body
		h.logger.Error("failed to encode register response", zap.Error(err))
	}
}

// Login handles login request-response requests.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidBody)
		return

	}
	if err := Validate.Struct(req); err != nil {
		var validationErrs validator.ValidationErrors

		if errors.As(err, &validationErrs) {
			if err := writeJSON(w, http.StatusBadRequest, toValidationErrorResponse(validationErrs)); err != nil {
				h.logger.Error("failed to encode validation error response", zap.Error(err))
			}
			return
		}

		writeError(w, http.StatusBadRequest, msgInvalidRequest)
		return

	}

	resp, err := h.service.Login(r.Context(), toLoginInput(req))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrInvalidPassword) {
			writeError(w, http.StatusUnauthorized, msgInvalidCredentials)
			return
		}

		writeError(w, http.StatusInternalServerError, msgInternalServer)
		h.logger.Error(msgInternalServer, zap.Error(err))
		return
	}

	setRefreshCookie(w, resp.TokenPair.RefreshToken)

	loginResp := &LoginResponse{
		AccessToken: resp.TokenPair.AccessToken,
		User:        toUserResponse(resp.User),
	}
	if err := writeJSON(w, http.StatusOK, loginResp); err != nil { //nolint:gosec // access token is intentionally returned to the client in the response body
		h.logger.Error("failed to encode login response", zap.Error(err))
	}
}

// RegisterPublicRoutes registers Handler's public routes
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
}

// RegisterProtectedRoutes registers Handler's protected routes
func (h *Handler) RegisterProtectedRoutes(_ chi.Router) {
	// to be implemented
}

// RegisterRefreshRoutes registers refresh cookie route which is particular to auth Handler
func (h *Handler) RegisterRefreshRoutes(r chi.Router) {
	r.Post("/auth/refresh", h.RefreshToken)
}

// RefreshToken extracts refresh token from cookie then refreshes an access token
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("refresh_token")

	if errors.Is(err, http.ErrNoCookie) {
		writeError(w, http.StatusUnauthorized, msgInvalidSession)
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, msgInternalServer)
		return
	}

	tokenPair, err := h.service.RefreshToken(r.Context(), cookie.Value)

	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			writeError(w, http.StatusUnauthorized, msgInvalidSession)
		} else {
			writeError(w, http.StatusInternalServerError, msgInternalServer)
		}

		return
	}

	setRefreshCookie(w, tokenPair.RefreshToken)

	response := &RefreshTokenResponse{
		AccessToken: tokenPair.AccessToken,
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil { //nolint:gosec // access token is intentionally returned to the client in the response body
		h.logger.Error("failed to encode refresh token response", zap.Error(err))
	}

}
