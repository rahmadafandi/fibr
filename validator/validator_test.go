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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStruct(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	t.Run("Valid", func(t *testing.T) {
		errs := ValidateStruct(&TestStruct{Name: "test", Email: "test@example.com"})
		assert.Empty(t, errs)
	})

	t.Run("InvalidUsesJSONFieldName", func(t *testing.T) {
		errs := ValidateStruct(&TestStruct{Name: "", Email: "bad"})
		assert.Len(t, errs, 2)
		fields := map[string]bool{}
		for _, e := range errs {
			fields[e.Field] = true
		}
		assert.True(t, fields["name"])
		assert.True(t, fields["email"])
	})
}

func TestErrorsToString(t *testing.T) {
	errs := []*ErrorResponse{
		{Field: "name", Tag: "required"},
		{Field: "email", Tag: "email"},
	}
	str := ErrorsToString(errs)
	assert.Equal(t, "field 'name' failed on the 'required' tag, field 'email' failed on the 'email' tag", str)
}

func TestRegisterCustomRule(t *testing.T) {
	err := Register("is_awesome", func(fl FieldLevel) bool {
		return fl.Field().String() == "awesome"
	})
	assert.NoError(t, err)

	type T struct {
		Word string `json:"word" validate:"is_awesome"`
	}
	assert.Empty(t, ValidateStruct(&T{Word: "awesome"}))
	assert.Len(t, ValidateStruct(&T{Word: "meh"}), 1)
}

func TestValidateNonStructDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		errs := ValidateStruct(nil)
		assert.NotEmpty(t, errs)
		assert.Equal(t, "invalid", errs[0].Tag)
	})
}
