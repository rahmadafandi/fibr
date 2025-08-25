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