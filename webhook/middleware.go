// Copyright 2026 Rahmad Afandi. MIT License.

package webhook

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/apierror"
)

// MiddlewareConfig configures the inbound-webhook verification middleware.
type MiddlewareConfig struct {
	Secret    string        // required shared secret
	Header    string        // signature header name; defaults to DefaultHeader
	Tolerance time.Duration // allowed clock skew; defaults to DefaultTolerance
}

// Middleware verifies the signature header on the request body before calling
// the next handler. On failure it returns a 401 via apierror (rendered by the
// bootstrap ErrorHandler). It does not reveal which check failed.
func Middleware(cfg MiddlewareConfig) fiber.Handler {
	if cfg.Header == "" {
		cfg.Header = DefaultHeader
	}
	if cfg.Tolerance == 0 {
		cfg.Tolerance = DefaultTolerance
	}
	return func(c *fiber.Ctx) error {
		if err := Verify(c.Body(), c.Get(cfg.Header), cfg.Secret, cfg.Tolerance); err != nil {
			return apierror.Unauthorized("invalid webhook signature").WithCode("invalid_signature")
		}
		return c.Next()
	}
}
