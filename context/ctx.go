package context

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

func GetContext(c *fiber.Ctx) context.Context {
	if ctx, ok := c.Locals("ctx").(context.Context); ok {
		return ctx
	}
	return context.Background()
}

func GetRequestID(c *fiber.Ctx) string {
	if requestID, ok := c.Locals("requestid").(string); ok {
		return requestID
	}
	return ""
}
