// Copyright 2026 Rahmad Afandi. MIT License.

package featureflag

import "github.com/gofiber/fiber/v2"

// localsKey is where the request-bound flags are stored.
const localsKey = "fibr_featureflag"

// bound pairs the Flags with the request's evaluation.
type bound struct {
	f *Flags
	e Eval
}

// Middleware stores the Flags and a per-request Eval (derived by extract) in the
// request locals, so handlers can call Enabled(c, flag). extract maps the
// request (auth, headers, ...) to an Eval; a nil extract uses an anonymous Eval.
func (f *Flags) Middleware(extract func(*fiber.Ctx) Eval) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var e Eval
		if extract != nil {
			e = extract(c)
		}
		c.Locals(localsKey, bound{f: f, e: e})
		return c.Next()
	}
}

// Enabled reports whether flag is on for the current request, using the Flags
// and Eval that Middleware stored. It returns false if the middleware is not
// installed.
func Enabled(c *fiber.Ctx, flag string) bool {
	b, ok := c.Locals(localsKey).(bound)
	if !ok {
		return false
	}
	return b.f.Enabled(c.UserContext(), flag, b.e)
}
