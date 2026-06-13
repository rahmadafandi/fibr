// Copyright 2026 Rahmad Afandi. MIT License.

// Package context provides request-scoped helpers: context access, request IDs, and type-safe locals.
package context

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// GetContext retrieves the context stored by ContextMiddleware from the Fiber
// request locals. It falls back to context.Background if no context was set.
func GetContext(c *fiber.Ctx) context.Context {
	if ctx, ok := c.Locals("ctx").(context.Context); ok {
		return ctx
	}
	return context.Background()
}

// GetRequestID retrieves the request ID stored by ContextMiddleware from the
// Fiber request locals. It returns an empty string if no request ID was set.
func GetRequestID(c *fiber.Ctx) string {
	if requestID, ok := c.Locals("requestid").(string); ok {
		return requestID
	}
	return ""
}

// SetLocal stores a value in the fiber context under key.
func SetLocal(c *fiber.Ctx, key string, value any) {
	c.Locals(key, value)
}

// GetLocal retrieves a value of type T from the fiber context.
// It returns the zero value of T if the key is absent or the type does not match.
func GetLocal[T any](c *fiber.Ctx, key string) T {
	if v, ok := c.Locals(key).(T); ok {
		return v
	}
	var zero T
	return zero
}
