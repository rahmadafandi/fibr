// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	fhcontext "github.com/rahmadafandi/fiber-helpers/context"
	"github.com/rahmadafandi/fiber-helpers/jwt"
	"github.com/rahmadafandi/fiber-helpers/response"
)

func bearerToken(c *fiber.Ctx) string {
	const prefix = "Bearer "
	h := c.Get("Authorization")
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return h[len(prefix):]
	}
	return ""
}

func parse(c *fiber.Ctx, secret string) (jwt.MapClaims, bool) {
	tok := bearerToken(c)
	if tok == "" {
		return nil, false
	}
	parsed, err := jwt.ValidateToken(tok, secret)
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, false
	}
	claims, err := jwt.ExtractClaimsFromJwt(parsed)
	if err != nil {
		return nil, false
	}
	if t, _ := claims["type"].(string); t == "refresh" {
		return nil, false // refresh tokens must not grant API access
	}
	return claims, true
}

// blocklisted reports whether the token's jti has been revoked via store. A
// token without a jti (not minted by an Issuer) is never blocklisted. The bool
// is true only on a confirmed block; a store error is returned so callers can
// decide their own fail-open/closed policy.
func blocklisted(ctx context.Context, store TokenStore, claims jwt.MapClaims) (bool, error) {
	if store == nil {
		return false, nil
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return false, nil
	}
	return store.IsBlocked(ctx, jti)
}

// RequireAuth extracts a bearer token, validates it with secret, stores the
// claims in the request context, then calls the next handler. A missing or
// invalid token gets a 401 response. Pass WithBlocklist to also reject revoked
// tokens by jti (fail-closed: a store error is treated as unauthorized). Tokens
// with no jti field (not minted by an Issuer) are not subject to the blocklist.
func RequireAuth(secret string, opts ...Option) fiber.Handler {
	cfg := newConfig(opts...)
	return func(c *fiber.Ctx) error {
		claims, ok := parse(c, secret)
		if !ok {
			return response.SendError(c, nil, "unauthorized", fiber.StatusUnauthorized)
		}
		if blocked, err := blocklisted(c.UserContext(), cfg.store, claims); err != nil || blocked {
			return response.SendError(c, nil, "unauthorized", fiber.StatusUnauthorized)
		}
		fhcontext.SetLocal(c, cfg.contextKey, claims)
		return c.Next()
	}
}

// Optional behaves like RequireAuth but proceeds without claims when the token
// is missing or invalid. With WithBlocklist, a blocked or errored token is
// treated as no auth (the handler runs anonymously) rather than rejected.
func Optional(secret string, opts ...Option) fiber.Handler {
	cfg := newConfig(opts...)
	return func(c *fiber.Ctx) error {
		if claims, ok := parse(c, secret); ok {
			if blocked, err := blocklisted(c.UserContext(), cfg.store, claims); err == nil && !blocked {
				fhcontext.SetLocal(c, cfg.contextKey, claims)
			}
		}
		return c.Next()
	}
}

// Claims returns the claims stored by the auth middleware.
func Claims(c *fiber.Ctx, opts ...Option) (jwt.MapClaims, bool) {
	cfg := newConfig(opts...)
	claims := fhcontext.GetLocal[jwt.MapClaims](c, cfg.contextKey)
	if claims == nil {
		return nil, false
	}
	return claims, true
}

// Subject returns claims["sub"] as a string ("" if absent or not a string).
func Subject(c *fiber.Ctx, opts ...Option) string {
	claims, ok := Claims(c, opts...)
	if !ok {
		return ""
	}
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}

// Scopes reads the "scopes" claim (a JSON array of strings) set when the token
// was issued; it normalizes the JWT-decoded []interface{} (or []string) form to
// []string. This library's RequireScope, HasScope, and Scopes all read that key.
func Scopes(c *fiber.Ctx, opts ...Option) []string {
	claims, ok := Claims(c, opts...)
	if !ok {
		return nil
	}
	return toStringSlice(claims["scopes"])
}

func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

// HasScope reports whether the request's claims include scope. It reads the
// same "scopes" claim as Scopes and RequireScope.
func HasScope(c *fiber.Ctx, scope string, opts ...Option) bool {
	for _, s := range Scopes(c, opts...) {
		if s == scope {
			return true
		}
	}
	return false
}

// RequireScope ensures the authenticated claims include scope (use after
// RequireAuth). It responds 401 when there are no claims and 403 when the scope
// is missing. If you pass WithContextKey to RequireAuth, pass the same option
// here so both use the same claims key.
func RequireScope(scope string, opts ...Option) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, ok := Claims(c, opts...); !ok {
			return response.SendError(c, nil, "unauthorized", fiber.StatusUnauthorized)
		}
		if !HasScope(c, scope, opts...) {
			return response.SendError(c, nil, "forbidden", fiber.StatusForbidden)
		}
		return c.Next()
	}
}
