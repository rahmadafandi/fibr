// Copyright 2026 Rahmad Afandi. MIT License.

package apikey_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rahmadafandi/fibr/apierror"
	"github.com/rahmadafandi/fibr/apikey"
)

func testApp(t *testing.T) (*fiber.App, string) {
	t.Helper()
	key, hash, err := apikey.Generate()
	require.NoError(t, err)

	a := apikey.New(apikey.Config{Store: apikey.MapStore(map[string]apikey.Identity{
		hash: {ID: "svc1", Scopes: []string{"read"}},
	})})

	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Use(a.Middleware())
	app.Get("/me", func(c *fiber.Ctx) error {
		id, _ := apikey.FromContext(c)
		return c.SendString(id.ID)
	})
	app.Get("/read", a.RequireScope("read"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/admin", a.RequireScope("admin"), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app, key
}

func status(t *testing.T, app *fiber.App, path, key string) int {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp.StatusCode
}

func TestMiddlewareRejectsAndAccepts(t *testing.T) {
	app, key := testApp(t)

	assert.Equal(t, fiber.StatusUnauthorized, status(t, app, "/me", ""))      // no key
	assert.Equal(t, fiber.StatusUnauthorized, status(t, app, "/me", "wrong")) // bad key
	assert.Equal(t, fiber.StatusOK, status(t, app, "/me", key))               // valid
}

func TestRequireScope(t *testing.T) {
	app, key := testApp(t)
	assert.Equal(t, fiber.StatusOK, status(t, app, "/read", key))         // has scope
	assert.Equal(t, fiber.StatusForbidden, status(t, app, "/admin", key)) // lacks scope
}

func TestIdentityAndHasScope(t *testing.T) {
	_, hash, err := apikey.Generate()
	require.NoError(t, err)
	_ = hash

	app := fiber.New()
	a := apikey.New(apikey.Config{Store: apikey.MapStore(map[string]apikey.Identity{
		apikey.Hash("k"): {ID: "u", Scopes: []string{"read"}},
	})})
	app.Use(a.Middleware())
	app.Get("/", func(c *fiber.Ctx) error {
		id, ok := apikey.FromContext(c)
		assert.True(t, ok)
		assert.Equal(t, "u", id.ID)
		assert.True(t, apikey.HasScope(c, "read"))
		assert.False(t, apikey.HasScope(c, "write"))
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "k")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
