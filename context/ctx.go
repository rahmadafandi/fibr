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

func CustomContext(c *fiber.Ctx, key string, value ...any) string {
	if len(value) == 0 {
		if v, ok := c.Locals(key).(string); ok {
			return v
		}
		return ""
	}
	if v, ok := c.Locals(key, value[0]).(string); ok {
		return v
	}
	return ""
}
