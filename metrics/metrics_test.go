// Copyright 2026 Rahmad Afandi. MIT License.

package metrics_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/metrics"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareRecordsAndHandlerScrapes(t *testing.T) {
	app := fiber.New()
	app.Use(metrics.Middleware())
	app.Get("/items/:id", func(c *fiber.Ctx) error { return c.SendString("ok") })
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
