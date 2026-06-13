// Copyright 2026 Rahmad Afandi. MIT License.

// Package middleware provides Fiber middleware: recover, request logging, request IDs, and auth.
package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/context"
	"github.com/rahmadafandi/fibr/logger"
	"github.com/rahmadafandi/fibr/response"
)

// Recover is a middleware that recovers from panics and logs the error.
func Recover(logger *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("%v", r)
				logger.Error(err, string(debug.Stack()), "request_id", context.GetRequestID(c))
				_ = c.Status(fiber.StatusInternalServerError).JSON(response.Response{
					Code:    fiber.StatusInternalServerError,
					Message: "Internal Server Error",
					Status:  "error",
				})
			}
		}()
		return c.Next()
	}
}
