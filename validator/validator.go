// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	stderrors "errors"
	"fmt"
	"reflect"
	"strings"

	govalidator "github.com/go-playground/validator/v10"
)

// Func is the signature for custom validation functions.
type Func = govalidator.Func

// FieldLevel is re-exported so callers can write custom rules without importing
// the underlying validator library directly.
type FieldLevel = govalidator.FieldLevel

// ErrorResponse represents a single validation error.
type ErrorResponse struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Param string `json:"param,omitempty"`
	Value string `json:"value,omitempty"`
}

var validate = newValidate()

func newValidate() *govalidator.Validate {
	v := govalidator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
	return v
}

// ValidateStruct validates a struct and returns a slice of ErrorResponse.
// If payload is nil or not a struct, it returns a single ErrorResponse with
// Tag "invalid" rather than panicking.
func ValidateStruct(payload interface{}) []*ErrorResponse {
	var result []*ErrorResponse
	err := validate.Struct(payload)
	if err == nil {
		return result
	}

	var ve govalidator.ValidationErrors
	if !stderrors.As(err, &ve) {
		return []*ErrorResponse{{Tag: "invalid", Value: err.Error()}}
	}

	for _, e := range ve {
		result = append(result, &ErrorResponse{
			Field: e.Field(),
			Tag:   e.Tag(),
			Param: e.Param(),
			Value: fmt.Sprintf("%v", e.Value()),
		})
	}
	return result
}

// Register adds a custom validation rule to the default validator.
//
// Register is NOT thread-safe and must be called during application startup,
// before any concurrent calls to ValidateStruct.
func Register(tag string, fn Func) error {
	return validate.RegisterValidation(tag, fn)
}

// ErrorsToString converts a slice of ErrorResponse to a single string.
func ErrorsToString(errs []*ErrorResponse) string {
	var s []string
	for _, err := range errs {
		s = append(s, fmt.Sprintf("field '%s' failed on the '%s' tag", err.Field, err.Tag))
	}
	return strings.Join(s, ", ")
}
