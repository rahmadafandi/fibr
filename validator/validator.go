package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// a single instance of the validator
var validate = validator.New()

// ErrorResponse represents a single validation error
type ErrorResponse struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

// ValidateStruct validates a struct and returns a slice of ErrorResponse
func ValidateStruct(payload interface{}) []*ErrorResponse {
	var errors []*ErrorResponse
	err := validate.Struct(payload)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.Field = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errors = append(errors, &element)
		}
	}
	return errors
}

// ErrorsToString converts a slice of ErrorResponse to a single string
func ErrorsToString(errs []*ErrorResponse) string {
	var s []string
	for _, err := range errs {
		s = append(s, fmt.Sprintf("field '%s' failed on the '%s' tag", err.Field, err.Tag))
	}
	return strings.Join(s, ", ")
}
