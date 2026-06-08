// Copyright 2026 Rahmad Afandi. MIT License.

// Command auth demonstrates password hashing and the JWT access/refresh token
// issuer backed by an in-memory revocation store. It needs no external
// services — run it with `go run ./auth`.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rahmadafandi/fiber-helpers/auth"
	"github.com/rahmadafandi/fiber-helpers/jwt"
)

func main() {
	ctx := context.Background()

	// 1. Hash a password at signup, verify it at login.
	hash, err := auth.Hash("s3cret-password")
	if err != nil {
		panic(err)
	}
	fmt.Println("password matches:", auth.Compare(hash, "s3cret-password") == nil)
	fmt.Println("wrong password:", auth.Compare(hash, "nope") == nil)

	// 2. Issue an access/refresh token pair for the authenticated user.
	issuer := auth.NewIssuer(
		"replace-with-a-real-secret",
		auth.NewMemoryStore(),
		auth.WithAccessTTL(15*time.Minute),
		auth.WithRefreshTTL(24*time.Hour),
	)

	pair, err := issuer.Issue(ctx, jwt.MapClaims{"sub": "user-1", "role": "admin"})
	if err != nil {
		panic(err)
	}
	fmt.Println("access token lifetime (s):", pair.ExpiresIn)

	// 3. Rotate the refresh token; the access token changes.
	rotated, err := issuer.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		panic(err)
	}
	fmt.Println("token rotated:", rotated.AccessToken != pair.AccessToken)

	// 4. Reusing the old refresh token is rejected (family revoked).
	_, err = issuer.Refresh(ctx, pair.RefreshToken)
	fmt.Println("reuse rejected:", err != nil)

	// 5. Logout revokes the current pair.
	if err := issuer.Logout(ctx, rotated.AccessToken, rotated.RefreshToken); err != nil {
		panic(err)
	}
	fmt.Println("logged out")
}
