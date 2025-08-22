package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/context"
	"github.com/rahmadafandi/fiber-helpers/logger"
)

// RequestLogger is a middleware that logs information about each request.
func RequestLogger(logger *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		stop := time.Now()

		logger.Info("request",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"latency", stop.Sub(start).String(),
			"request_id", context.GetRequestID(c),
		)

		return err
	}
}
