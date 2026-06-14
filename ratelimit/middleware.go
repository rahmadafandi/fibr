// Copyright 2026 Rahmad Afandi. MIT License.

package ratelimit

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/rahmadafandi/fibr/logger"
)

// Middleware limits requests per key under rule (cost 1 each). keyFunc derives
// the bucket key; when nil it uses the client IP. On a deny it responds 429 with
// Retry-After and X-RateLimit-* headers. A Redis error fails open (the request
// is allowed) so the limiter cannot take the app down.
func (l *Limiter) Middleware(rule Rule, keyFunc func(*fiber.Ctx) string) fiber.Handler {
	if keyFunc == nil {
		keyFunc = func(c *fiber.Ctx) string { return c.IP() }
	}
	log := logger.Default()
	limit := strconv.Itoa(rule.Capacity)

	return func(c *fiber.Ctx) error {
		res, err := l.Allow(c.UserContext(), keyFunc(c), rule, 1)
		if err != nil {
			log.Error(err, "ratelimit: allow failed; failing open")
			return c.Next()
		}

		c.Set("X-RateLimit-Limit", limit)
		c.Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))

		if !res.Allowed {
			c.Set("Retry-After", strconv.Itoa(int(res.RetryAfter.Seconds())))
			return c.SendStatus(fiber.StatusTooManyRequests)
		}
		return c.Next()
	}
}
