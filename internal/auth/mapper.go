package auth

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// toRegisterInput maps a RegisterRequest to the service-layer RegisterInput.
func toRegisterInput(req RegisterRequest) RegisterInput {
	return RegisterInput(req)
}

// toLoginInput maps a LoginRequest to the service-layer LoginInput.
func toLoginInput(req LoginRequest) LoginInput {
	return LoginInput(req)
}

// toUserResponse maps a domain User to its client-safe projection.
func toUserResponse(u User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: u.CreatedAt,
	}
}

// toValidationErrorResponse maps validator field errors to the API's error shape.
func toValidationErrorResponse(errs validator.ValidationErrors) ValidationErrorResponse {
	resp := ValidationErrorResponse{}
	for _, fe := range errs {
		resp.Errors = append(resp.Errors, FieldError{
			Field:   fe.Field(),
			Message: fmt.Sprintf("failed on the '%s' rule", fe.Tag()),
		})
	}
	return resp
}
