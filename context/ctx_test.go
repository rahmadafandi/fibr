// Copyright 2026 Rahmad Afandi. MIT License.

package context

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestLocals(t *testing.T) {
	app := fiber.New()

	t.Run("SetAndGetString", func(t *testing.T) {
		app.Get("/str", func(c *fiber.Ctx) error {
			SetLocal(c, "k", "value")
			assert.Equal(t, "value", GetLocal[string](c, "k"))
			return nil
		})
		resp, err := app.Test(httptest.NewRequest("GET", "/str", nil))
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("MissingReturnsZero", func(t *testing.T) {
		app.Get("/missing", func(c *fiber.Ctx) error {
			assert.Equal(t, "", GetLocal[string](c, "absent"))
			assert.Equal(t, 0, GetLocal[int](c, "absent"))
			return nil
		})
		resp, err := app.Test(httptest.NewRequest("GET", "/missing", nil))
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("SetAndGetStruct", func(t *testing.T) {
		type payload struct{ Name string }
		app.Get("/struct", func(c *fiber.Ctx) error {
			SetLocal(c, "p", payload{Name: "x"})
			assert.Equal(t, payload{Name: "x"}, GetLocal[payload](c, "p"))
			return nil
		})
		resp, err := app.Test(httptest.NewRequest("GET", "/struct", nil))
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
