// Copyright 2026 Rahmad Afandi. MIT License.

package jwt_test

import (
	"fmt"

	"github.com/rahmadafandi/fiber-helpers/jwt"
)

// Sign a token, validate it with the same secret, then read a claim back out.
func ExampleGenerateToken() {
	secret := "super-secret-key"

	token, err := jwt.GenerateToken(jwt.MapClaims{"sub": "user-42", "role": "admin"}, secret)
	if err != nil {
		panic(err)
	}

	parsed, err := jwt.ValidateToken(token, secret)
	if err != nil {
		panic(err)
	}

	claims, err := jwt.ExtractClaimsFromJwt(parsed)
	if err != nil {
		panic(err)
	}

	fmt.Println("sub:", claims["sub"])
	fmt.Println("role:", claims["role"])
	// Output:
	// sub: user-42
	// role: admin
}

// Validating with the wrong secret fails.
func ExampleValidateToken_wrongSecret() {
	token, _ := jwt.GenerateToken(jwt.MapClaims{"sub": "user-42"}, "right-secret")

	_, err := jwt.ValidateToken(token, "wrong-secret")
	fmt.Println("valid:", err == nil)
	// Output:
	// valid: false
}
