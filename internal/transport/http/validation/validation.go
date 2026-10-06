// Package validation provides a shared validator instance for request DTOs across domains.
package validation

import "github.com/go-playground/validator/v10"

// Validate is a shared, reusable validator instance for all domains' request DTOs.
var Validate = validator.New()
