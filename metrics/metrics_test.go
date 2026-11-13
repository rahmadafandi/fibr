// Copyright 2026 Rahmad Afandi. MIT License.

package metrics_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/rahmadafandi/fibr/v2/apierror"
	"github.com/rahmadafandi/fibr/v2/metrics"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareRecordsAndHandlerScrapes(t *testing.T) {
	app := fiber.New()
	app.Use(metrics.Middleware())
	app.Get("/items/:id", func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get(metrics.MetricsPath, metrics.Handler())

	resp, err := app.Test(httptest.NewRequest("GET", "/items/42", nil))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest("GET", "/metrics", nil))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	require.Contains(t, s, "http_requests_total")
	require.Contains(t, s, `path="/items/:id"`)
	require.Contains(t, s, `status="200"`)
	require.Contains(t, s, "http_request_duration_seconds")
	require.Contains(t, s, "go_goroutines")
	require.NotContains(t, s, `path="/metrics"`)
}

func TestMiddlewareRecordsErrorStatus(t *testing.T) {
	app := fiber.New()
	app.Use(metrics.Middleware())
	app.Get("/boom", func(c fiber.Ctx) error { return fiber.NewError(503, "down") })
	app.Get(metrics.MetricsPath, metrics.Handler())

	resp, err := app.Test(httptest.NewRequest("GET", "/boom", nil))
	require.NoError(t, err)
	require.Equal(t, 503, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest("GET", "/metrics", nil))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), `path="/boom",status="503"`)
}

func TestMiddlewareRecordsAPIErrorStatus(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Use(metrics.Middleware())
	app.Get("/missing", func(c fiber.Ctx) error { return apierror.NotFound("nope") })
	app.Get(metrics.MetricsPath, metrics.Handler())

	resp, err := app.Test(httptest.NewRequest("GET", "/missing", nil))
	require.NoError(t, err)
	require.Equal(t, 404, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest("GET", "/metrics", nil))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), `path="/missing",status="404"`)
}

func TestMiddlewareLabelsUnmatchedRoutes(t *testing.T) {
	app := fiber.New()
	app.Use(metrics.Middleware())
	app.Get("/", func(c fiber.Ctx) error { return c.SendString("home") })
	app.Get(metrics.MetricsPath, metrics.Handler())

	for _, p := range []string{"/", "/no-such-route", "/another/missing"} {
		_, err := app.Test(httptest.NewRequest("GET", p, nil))
		require.NoError(t, err)
	}

	resp, err := app.Test(httptest.NewRequest("GET", "/metrics", nil))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	out := string(body)
	require.Contains(t, out, `http_requests_total{method="GET",path="unmatched",status="404"}`)
	require.Contains(t, out, `http_requests_total{method="GET",path="/",status="200"}`)
	require.NotContains(t, out, `path="/",status="404"`)
}
