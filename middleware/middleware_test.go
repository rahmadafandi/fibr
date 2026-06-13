// Copyright 2026 Rahmad Afandi. MIT License.

package middleware

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestMiddleware(t *testing.T) {
	app := fiber.New()
	log := logger.New(&bytes.Buffer{}, zerolog.InfoLevel)

	t.Run("Recover", func(t *testing.T) {
		app.Get("/recover", Recover(log), func(c *fiber.Ctx) error {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/recover", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("RequestLogger", func(t *testing.T) {
		app.Get("/logger", RequestLogger(log), func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/logger", nil)
		app.Test(req)
	})
}

func TestContextMiddlewareSetsRequestIDHeader(t *testing.T) {
	app := fiber.New()
	app.Use(ContextMiddleware(time.Second))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

func TestRequestLoggerAddsTraceIDs(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, zerolog.InfoLevel)

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		ctx, span := tp.Tracer("test").Start(c.UserContext(), "req")
		defer span.End()
		c.SetUserContext(ctx)
		return c.Next()
	})
	app.Use(RequestLogger(log))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	_, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
	require.Contains(t, buf.String(), "trace_id")
	require.Contains(t, buf.String(), "span_id")
}

func TestRequestLoggerNoTraceIDsWithoutSpan(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, zerolog.InfoLevel)
	app := fiber.New()
	app.Use(RequestLogger(log))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	_, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
	require.NotContains(t, buf.String(), "trace_id")
}
