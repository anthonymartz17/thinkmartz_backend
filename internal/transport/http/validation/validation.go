// Package validation provides a shared validator instance for request DTOs across domains.
package validation

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// Validate is a shared, reusable validator instance for all domains' request DTOs.
var Validate = validator.New()

// FieldError represents a single validation failure on one field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse is the JSON body returned when request
// validation fails.
type ErrorResponse struct {
	Errors []FieldError `json:"errors"`
}

// ToErrorResponse maps validator field errors to the API's error shape.
func ToErrorResponse(errs validator.ValidationErrors) ErrorResponse {
	resp := ErrorResponse{}
	for _, fe := range errs {
		resp.Errors = append(resp.Errors, FieldError{
			Field:   fe.Field(),
			Message: fmt.Sprintf("failed on the '%s' rule", fe.Tag()),
		})
	}
	return resp
}
