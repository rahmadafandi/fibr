// Copyright 2026 Rahmad Afandi. MIT License.

package featureflag_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/featureflag"
)

func bodyOf(t *testing.T, app *fiber.App, userHeader string) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	if userHeader != "" {
		req.Header.Set("X-User", userHeader)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

func TestFiberMiddlewareAndEnabled(t *testing.T) {
	f := featureflag.New(featureflag.Rules(map[string]featureflag.Rule{
		"beta": {Users: []string{"u1"}},
	}))

	app := fiber.New()
	app.Use(f.Middleware(func(c *fiber.Ctx) featureflag.Eval {
		return featureflag.Eval{UserID: c.Get("X-User")}
	}))
	app.Get("/", func(c *fiber.Ctx) error {
		if featureflag.Enabled(c, "beta") {
			return c.SendString("beta")
		}
		return c.SendString("stable")
	})

	assert.Equal(t, "beta", bodyOf(t, app, "u1"))
	assert.Equal(t, "stable", bodyOf(t, app, "u2"))
}

func TestEnabledWithoutMiddleware(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		assert.False(t, featureflag.Enabled(c, "anything"))
		return c.SendStatus(fiber.StatusOK)
	})
	_, err := app.Test(httptest.NewRequest("GET", "/", nil))
	require.NoError(t, err)
}
