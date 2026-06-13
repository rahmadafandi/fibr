// Copyright 2026 Rahmad Afandi. MIT License.

package jwt_test

import (
	"testing"

	"github.com/rahmadafandi/fibr/jwt"
)

// FuzzValidateToken ensures ValidateToken never panics on arbitrary token
// strings — malformed tokens must return an error, not crash.
func FuzzValidateToken(f *testing.F) {
	f.Add("", "secret")
	f.Add("not.a.jwt", "secret")
	f.Add("a.b.c", "")
	f.Add("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.x", "secret")

	f.Fuzz(func(t *testing.T, token, secret string) {
		_, _ = jwt.ValidateToken(token, secret) // must not panic
	})
}
