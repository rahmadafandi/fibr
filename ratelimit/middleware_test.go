// Copyright 2026 Rahmad Afandi. MIT License.

package ratelimit

import (
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareLimitsAndHeaders(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	l := New(client)

	app := fiber.New()
	app.Use(l.Middleware(Rule{Capacity: 2, RefillPerSec: 1}, func(c *fiber.Ctx) string {
		return "fixed" // same bucket for every request
	}))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	// First two pass.
	for range 2 {
		resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.Equal(t, "2", resp.Header.Get("X-RateLimit-Limit"))
	}

	// Third is rejected with 429 + Retry-After.
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Retry-After"))
	assert.Equal(t, "0", resp.Header.Get("X-RateLimit-Remaining"))
}
