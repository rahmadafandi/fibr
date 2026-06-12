// Copyright 2026 Rahmad Afandi. MIT License.

package bootstrap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/apierror"
	"github.com/rahmadafandi/fibr/database"
	"github.com/rahmadafandi/fibr/health"
	"github.com/rahmadafandi/fibr/i18n"
	"github.com/rahmadafandi/fibr/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewRoutesWork(t *testing.T) {
	app := New(Options{})
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendString("pong") })

	resp, err := app.Test(httptest.NewRequest("GET", "/ping", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestNewWiresHealth(t *testing.T) {
	app := New(Options{
		HealthChecks: []health.NamedCheck{
			health.Check("ok", func(ctx context.Context) error { return nil }),
		},
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/livez", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest("GET", "/readyz", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestNewSetsRequestIDHeader(t *testing.T) {
	app := New(Options{})
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

func TestNewDBAddsCleanup(t *testing.T) {
	db, err := database.NewBun("file::memory:?cache=shared")
	require.NoError(t, err)
	app := New(Options{DB: db})
	assert.Len(t, app.cleanup, 1)
}

func TestNewNoDBNoCleanup(t *testing.T) {
	app := New(Options{})
	assert.Empty(t, app.cleanup)
}

func TestHealthExemptFromRateLimit(t *testing.T) {
	app := New(Options{
		RateLimit: 1,
		HealthChecks: []health.NamedCheck{
			health.Check("ok", func(ctx context.Context) error { return nil }),
		},
	})
	// Probe hit far more than the limit must never be throttled.
	for i := 0; i < 5; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/livez", nil))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}
}

func TestMetricsOptInServesScrape(t *testing.T) {
	app := New(Options{Metrics: true})
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })

	_, err := app.Test(httptest.NewRequest("GET", "/x", nil))
	require.NoError(t, err)

	resp, err := app.Test(httptest.NewRequest("GET", "/metrics", nil))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), "http_requests_total")
}

func TestMetricsOffNoScrape(t *testing.T) {
	app := New(Options{})
	resp, err := app.Test(httptest.NewRequest("GET", "/metrics", nil))
	require.NoError(t, err)
	require.Equal(t, 404, resp.StatusCode)
}

func TestTracingOptInRecordsSpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)

	app := New(Options{Tracing: true})
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })

	_, err := app.Test(httptest.NewRequest("GET", "/x", nil))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(sr.Ended()), 1, "otelfiber should record a server span")
}

func TestAsynqmonMountServes(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("mon-ok"))
	})
	app := New(Options{Asynqmon: &AsynqmonMount{Handler: h, Path: "/monitoring"}})

	resp, err := app.Test(httptest.NewRequest("GET", "/monitoring", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "mon-ok", string(body))
}

func TestAsynqmonMiddlewareApplied(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	guard := func(c *fiber.Ctx) error { return c.SendStatus(401) }
	app := New(Options{Asynqmon: &AsynqmonMount{Handler: h, Path: "/monitoring", Middleware: []fiber.Handler{guard}}})

	resp, err := app.Test(httptest.NewRequest("GET", "/monitoring", nil))
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestCleanupHooksRegistered(t *testing.T) {
	called := false
	app := New(Options{Cleanup: []func(context.Context) error{
		func(context.Context) error { called = true; return nil },
	}})
	for _, fn := range app.cleanup {
		_ = fn(context.Background())
	}
	require.True(t, called, "Options.Cleanup hooks must be registered as cleanup")
}

func TestDefaultErrorHandlerRendersAPIError(t *testing.T) {
	app := New(Options{})
	app.Get("/boom", func(c *fiber.Ctx) error {
		return apierror.NotFound("nope").WithCode("gone")
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/boom", nil))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 404, resp.StatusCode)
	assert.Contains(t, string(body), `"error":"gone"`)
}

func TestUserErrorHandlerNotOverridden(t *testing.T) {
	sentinel := func(c *fiber.Ctx, err error) error {
		return c.Status(418).SendString("teapot")
	}
	app := New(Options{FiberConfig: fiber.Config{ErrorHandler: sentinel}})
	app.Get("/boom", func(c *fiber.Ctx) error {
		return apierror.NotFound("nope")
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/boom", nil))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 418, resp.StatusCode)
	assert.Equal(t, "teapot", string(body))
}

func TestSecurityHeaders(t *testing.T) {
	app := New(Options{SecurityHeaders: true})
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
}

func TestNoSecurityHeadersByDefault(t *testing.T) {
	app := New(Options{})
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
	assert.Empty(t, resp.Header.Get("X-Content-Type-Options"))
}

func TestCompression(t *testing.T) {
	app := New(Options{Compression: true})
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString(strings.Repeat("a", 2048)) })
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
}

func TestIdempotencyReplaysResponse(t *testing.T) {
	app := New(Options{Idempotency: true})
	var calls int
	app.Post("/pay", func(c *fiber.Ctx) error {
		calls++
		return c.SendString(strconv.Itoa(calls))
	})

	key := "11111111-1111-1111-1111-111111111111" // 36-char UUID
	do := func() string {
		req := httptest.NewRequest("POST", "/pay", nil)
		req.Header.Set("X-Idempotency-Key", key)
		resp, err := app.Test(req)
		require.NoError(t, err)
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}

	first := do()
	second := do()
	assert.Equal(t, "1", first)
	assert.Equal(t, "1", second) // replayed cached response
	assert.Equal(t, 1, calls)    // handler ran once
}

func TestIdempotencyNoKeyPassesThrough(t *testing.T) {
	app := New(Options{Idempotency: true})
	var calls int
	app.Post("/pay", func(c *fiber.Ctx) error {
		calls++
		return c.SendString(strconv.Itoa(calls))
	})
	for i := 0; i < 3; i++ {
		resp, err := app.Test(httptest.NewRequest("POST", "/pay", nil))
		require.NoError(t, err)
		_ = resp
	}
	assert.Equal(t, 3, calls)
}

func TestOpenAPIMounted(t *testing.T) {
	oapi := openapi.New(openapi.Info{Title: "T", Version: "1.0.0"})
	oapi.Register("GET", "/ping", openapi.Op{Summary: "ping"})

	app := New(Options{OpenAPI: oapi})

	specResp, err := app.Test(httptest.NewRequest("GET", "/openapi.json", nil))
	require.NoError(t, err)
	require.Equal(t, 200, specResp.StatusCode)

	docsResp, err := app.Test(httptest.NewRequest("GET", "/docs", nil))
	require.NoError(t, err)
	require.Equal(t, 200, docsResp.StatusCode)
}

func TestI18nMiddlewareMounted(t *testing.T) {
	b := i18n.New(i18n.WithFallback("en"))
	b.LoadMap("en", map[string]any{"welcome": "Hello"})
	b.LoadMap("id", map[string]any{"welcome": "Halo"})

	app := New(Options{I18n: b})
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString(i18n.T(c, "welcome", nil)) })

	req := httptest.NewRequest("GET", "/x?lang=id", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, "Halo", string(body))
}
