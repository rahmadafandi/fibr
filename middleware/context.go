package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
)

func ContextMiddleware(timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), timeout)
		defer cancel()
		if requestID, ok := c.Locals("requestid").(string); ok && requestID != "" {
			ctx = context.WithValue(ctx, RequestIDKey, requestID)
		} else {
			requestID = uuid.New().String()
			ctx = context.WithValue(ctx, RequestIDKey, requestID)
			c.Locals("requestid", requestID)
		}
		c.Locals("ctx", ctx)
		return c.Next()
	}
}
