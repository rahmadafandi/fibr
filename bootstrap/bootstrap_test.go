// Copyright 2026 Rahmad Afandi. MIT License.

package bootstrap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/database"
	"github.com/rahmadafandi/fiber-helpers/health"
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
