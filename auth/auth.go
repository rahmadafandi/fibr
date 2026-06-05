// Copyright 2026 Rahmad Afandi. MIT License.

// Package auth provides bcrypt password hashing and JWT bearer authentication
// middleware (with scope checks) for Fiber apps, built on the jwt, context, and
// response packages.
package auth

import "golang.org/x/crypto/bcrypt"

// DefaultClaimsKey is the Fiber locals key under which the auth middleware
// stores validated claims.
const DefaultClaimsKey = "auth_claims"

// Hash returns the bcrypt hash of password using the default cost. Note bcrypt
// only considers the first 72 bytes of the input.
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare reports whether password matches the bcrypt hash; it returns nil on a
// match and a non-nil error otherwise.
func Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// Option configures the auth middleware and accessors.
type Option func(*config)

type config struct {
	contextKey string
	store      TokenStore
}

func newConfig(opts ...Option) config {
	c := config{contextKey: DefaultClaimsKey}
	for _, o := range opts {
		o(&c)
	}
	return c
}

// WithContextKey sets the Fiber locals key used to store and read claims
// (default DefaultClaimsKey).
func WithContextKey(key string) Option {
	return func(c *config) {
		if key != "" {
			c.contextKey = key
		}
	}
}

// WithBlocklist makes RequireAuth and Optional reject access tokens whose jti
// has been revoked via store (e.g. by Issuer.Logout or Issuer.Refresh). Tokens
// without a jti (not minted by an Issuer) are not affected. Without this option
// the middleware performs no store lookups.
func WithBlocklist(store TokenStore) Option {
	return func(c *config) {
		c.store = store
	}
}
