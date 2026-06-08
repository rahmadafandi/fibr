// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/response"
)

// ActiveTeam returns the active team id from the "team" claim ("" if absent or
// not a string).
func ActiveTeam(c *fiber.Ctx, opts ...Option) string {
	claims, ok := Claims(c, opts...)
	if !ok {
		return ""
	}
	if t, ok := claims["team"].(string); ok {
		return t
	}
	return ""
}

// TeamRole returns the caller's role in the active team from the "role" claim
// ("" if absent or not a string).
func TeamRole(c *fiber.Ctx, opts ...Option) string {
	claims, ok := Claims(c, opts...)
	if !ok {
		return ""
	}
	if r, ok := claims["role"].(string); ok {
		return r
	}
	return ""
}

// RequireTeam ensures an active team is present (use after RequireAuth). It
// responds 401 when there are no claims and 403 when no team is active.
func RequireTeam(opts ...Option) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, ok := Claims(c, opts...); !ok {
			return response.SendError(c, nil, "unauthorized", fiber.StatusUnauthorized)
		}
		if ActiveTeam(c, opts...) == "" {
			return response.SendError(c, nil, "no active team", fiber.StatusForbidden)
		}
		return c.Next()
	}
}

// RequireRole ensures the caller's active-team role is one of roles (use after
// RequireAuth). It responds 401 when there are no claims and 403 when the role
// does not match. RequireRole reads the default claims context key; if you use
// WithContextKey, gate with permission checks (RequireScope) instead.
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, ok := Claims(c); !ok {
			return response.SendError(c, nil, "unauthorized", fiber.StatusUnauthorized)
		}
		role := TeamRole(c)
		for _, r := range roles {
			if r == role {
				return c.Next()
			}
		}
		return response.SendError(c, nil, "forbidden", fiber.StatusForbidden)
	}
}
