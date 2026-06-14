// Copyright 2026 Rahmad Afandi. MIT License.

package apikey

import (
	"slices"

	"github.com/gofiber/fiber/v2"

	"github.com/rahmadafandi/fibr/apierror"
)

const localsKey = "fibr_apikey_identity"

const defaultHeader = "X-API-Key"

// Config configures an Authenticator.
type Config struct {
	Store  Store
	Header string // default "X-API-Key"
}

// Authenticator validates API keys against a Store.
type Authenticator struct {
	store  Store
	header string
}

// New returns an Authenticator. Store is required.
func New(cfg Config) *Authenticator {
	header := cfg.Header
	if header == "" {
		header = defaultHeader
	}
	return &Authenticator{store: cfg.Store, header: header}
}

// Middleware reads the key from the configured header, hashes it, looks it up,
// and on success stores the Identity in locals and calls Next; otherwise it
// returns 401 (or 500 on a store error).
func (a *Authenticator) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get(a.header)
		if key == "" {
			return apierror.Unauthorized("missing API key")
		}
		id, err := a.store.Lookup(c.UserContext(), Hash(key))
		if err != nil {
			return apierror.Internal("api key lookup failed")
		}
		if id == nil {
			return apierror.Unauthorized("invalid API key")
		}
		c.Locals(localsKey, id)
		return c.Next()
	}
}

// RequireScope is middleware that returns 403 unless the request's Identity has
// scope. It must run after Middleware.
func (a *Authenticator) RequireScope(scope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, ok := FromContext(c)
		if !ok {
			return apierror.Unauthorized("not authenticated")
		}
		if !slices.Contains(id.Scopes, scope) {
			return apierror.Forbidden("missing required scope: " + scope)
		}
		return c.Next()
	}
}

// FromContext returns the authenticated identity stored by Middleware.
func FromContext(c *fiber.Ctx) (*Identity, bool) {
	id, ok := c.Locals(localsKey).(*Identity)
	return id, ok
}

// HasScope reports whether the request's Identity has scope.
func HasScope(c *fiber.Ctx, scope string) bool {
	id, ok := FromContext(c)
	return ok && slices.Contains(id.Scopes, scope)
}
