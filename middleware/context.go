// Copyright 2026 Rahmad Afandi. MIT License.

package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type contextKey string

const (
	// RequestIDKey is the context key used to store and retrieve the request ID.
	RequestIDKey contextKey = "request_id"
)

// ContextMiddleware attaches a timeout-bounded context and a unique request ID
// to every incoming Fiber request. The context is derived from c.UserContext()
// (so it keeps values set by earlier middleware, such as a tracing span) and is
// available both from context.GetContext and c.UserContext().
func ContextMiddleware(timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), timeout)
		defer cancel()
		requestID, ok := c.Locals("requestid").(string)
		if !ok || requestID == "" {
			requestID = uuid.New().String()
			c.Locals("requestid", requestID)
		}
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
		c.Locals("ctx", ctx)
		c.SetUserContext(ctx)
		c.Set("X-Request-ID", requestID)
		return c.Next()
	}
}
