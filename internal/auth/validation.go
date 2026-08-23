package auth

import "github.com/go-playground/validator/v10"

// Validate is a shared, reusable validator instance for all auth request
// DTOs across the package's handlers.
var Validate = validator.New()
