// Copyright 2026 Rahmad Afandi. MIT License.

package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/context"
	"github.com/rahmadafandi/fiber-helpers/logger"
	"github.com/rahmadafandi/fiber-helpers/response"
)

// Recover is a middleware that recovers from panics and logs the error.
func Recover(logger *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("%v", r)
				logger.Error(err, string(debug.Stack()), "request_id", context.GetRequestID(c))
				c.Status(fiber.StatusInternalServerError).JSON(response.Response{
					Code:    fiber.StatusInternalServerError,
					Message: "Internal Server Error",
					Status:  "error",
				})
			}
		}()
		return c.Next()
	}
}
