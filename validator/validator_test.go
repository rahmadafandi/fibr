package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStruct(t *testing.T) {
	type TestStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	t.Run("Valid", func(t *testing.T) {
		payload := &TestStruct{
			Name:  "test",
			Email: "test@example.com",
		}

		errs := ValidateStruct(payload)
		assert.Empty(t, errs)
	})

	t.Run("Invalid", func(t *testing.T) {
		payload := &TestStruct{
			Name:  "",
			Email: "test",
		}

		errs := ValidateStruct(payload)
		assert.Len(t, errs, 2)
	})
}

func TestErrorsToString(t *testing.T) {
	errs := []*ErrorResponse{
		{Field: "Name", Tag: "required"},
		{Field: "Email", Tag: "email"},
	}

	str := ErrorsToString(errs)
	assert.Equal(t, "field 'Name' failed on the 'required' tag, field 'Email' failed on the 'email' tag", str)
}
