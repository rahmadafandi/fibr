// Copyright 2026 Rahmad Afandi. MIT License.

package fibrtest

import "github.com/rahmadafandi/fibr/jwt"

// Token mints a signed JWT with the given claims, for testing authed routes
// without going through a login flow. A signing error calls Fatalf.
func Token(t TB, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok, err := jwt.GenerateToken(claims, secret)
	if err != nil {
		t.Fatalf("fibrtest: generate token: %v", err)
		return ""
	}
	return tok
}
