// Copyright 2026 Rahmad Afandi. MIT License.

package bootstrap

import (
	"io"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"

	"github.com/rahmadafandi/fibr/v2/logger"
)

// benchRequest drives the app's fasthttp handler directly, so the numbers are
// the cost of the middleware chain and not of a test client.
func benchRequest(b *testing.B, handler fasthttp.RequestHandler) {
	b.Helper()
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fiber.MethodGet)
	ctx.Request.SetRequestURI("/hello")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		handler(ctx)
	}
	if code := ctx.Response.StatusCode(); code != fiber.StatusOK {
		b.Fatalf("status = %d", code)
	}
}

func hello(c fiber.Ctx) error { return c.SendString("hello") }

// BenchmarkRequestPlainFiber is the baseline: a bare Fiber app.
func BenchmarkRequestPlainFiber(b *testing.B) {
	app := fiber.New()
	app.Get("/hello", hello)
	benchRequest(b, app.Handler())
}

// BenchmarkRequestBootstrapDefault is bootstrap.New with no optional features:
// recover, request context, request logging, health.
func BenchmarkRequestBootstrapDefault(b *testing.B) {
	app := New(Options{Logger: logger.New(io.Discard, logger.InfoLevel)})
	app.Get("/hello", hello)
	benchRequest(b, app.Handler())
}

// BenchmarkRequestBootstrapFull turns on the optional per-request middleware
// that needs no external service.
func BenchmarkRequestBootstrapFull(b *testing.B) {
	app := New(Options{
		Logger:          logger.New(io.Discard, logger.InfoLevel),
		Metrics:         true,
		SecurityHeaders: true,
		Compression:     true,
		CORSOrigins:     []string{"https://app.example.com"},
		RateLimit:       1 << 30,
	})
	app.Get("/hello", hello)
	benchRequest(b, app.Handler())
}
