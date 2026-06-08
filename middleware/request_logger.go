// Copyright 2026 Rahmad Afandi. MIT License.

package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/context"
	"github.com/rahmadafandi/fibr/logger"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// RequestLogger is a middleware that logs information about each request. When
// the request carries a valid OTel span context, it adds trace_id and span_id
// fields so logs can be correlated with traces.
func RequestLogger(logger *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		stop := time.Now()

		fields := []interface{}{
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"latency", stop.Sub(start).String(),
			"request_id", context.GetRequestID(c),
		}
		if sc := oteltrace.SpanContextFromContext(c.UserContext()); sc.IsValid() {
			fields = append(fields, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
		}
		logger.Info("request", fields...)

		return err
	}
}
