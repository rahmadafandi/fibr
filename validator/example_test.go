// Copyright 2026 Rahmad Afandi. MIT License.

package validator_test

import (
	"fmt"

	"github.com/rahmadafandi/fibr/validator"
)

// ValidateStruct checks `validate` struct tags and returns one ErrorResponse
// per failing field. Field names come from the `json` tag.
func ExampleValidateStruct() {
	type SignupInput struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}

	errs := validator.ValidateStruct(SignupInput{Email: "not-an-email", Password: "short"})
	for _, e := range errs {
		fmt.Printf("%s failed %q\n", e.Field, e.Tag)
	}
	// Output:
	// email failed "email"
	// password failed "min"
}

// A valid struct returns no errors.
func ExampleValidateStruct_valid() {
	type SignupInput struct {
		Email string `json:"email" validate:"required,email"`
	}

	errs := validator.ValidateStruct(SignupInput{Email: "user@example.com"})
	fmt.Println("errors:", len(errs))
	// Output:
	// errors: 0
}
