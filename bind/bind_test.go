// Copyright 2026 Rahmad Afandi. MIT License.

package bind_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/bind"
	"github.com/stretchr/testify/require"
)

type createInput struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

func bodyOf(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

func TestBodyValid(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		in, ok := bind.Body[createInput](c)
		if !ok {
			return nil
		}
		return c.SendString(in.Name)
	})

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"ada","email":"ada@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "ada", bodyOf(t, resp))
}

func TestBodyMalformed(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		_, ok := bind.Body[createInput](c)
		if !ok {
			return nil
		}
		return c.SendString("unreachable")
	})

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{not json`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 400, resp.StatusCode)
	require.Contains(t, bodyOf(t, resp), `"status":"error"`)
}

func TestBodyValidationFails(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		_, ok := bind.Body[createInput](c)
		if !ok {
			return nil
		}
		return c.SendString("unreachable")
	})

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"","email":"nope"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 422, resp.StatusCode)
	body := bodyOf(t, resp)
	require.Contains(t, body, `"status":"error"`)
	require.Contains(t, body, `"field":"name"`)
	require.Contains(t, body, `"field":"email"`)
}

type filterInput struct {
	Sort string `query:"sort" validate:"required,oneof=name date"`
}

func TestQueryValidAndInvalid(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		in, ok := bind.Query[filterInput](c)
		if !ok {
			return nil
		}
		return c.SendString(in.Sort)
	})

	ok, err := app.Test(httptest.NewRequest("GET", "/?sort=name", nil))
	require.NoError(t, err)
	require.Equal(t, 200, ok.StatusCode)
	require.Equal(t, "name", bodyOf(t, ok))

	bad, err := app.Test(httptest.NewRequest("GET", "/?sort=bogus", nil))
	require.NoError(t, err)
	require.Equal(t, 422, bad.StatusCode)
	require.Contains(t, bodyOf(t, bad), `"status":"error"`)
}

// min=1 (not required): a route id is always present but `required` on an int
// means "non-zero", which would 422 a legitimate /items/0. min=1 is the intent.
type idParam struct {
	ID int `params:"id" validate:"min=1"`
}

func TestParamsValidAndInvalid(t *testing.T) {
	app := fiber.New()
	app.Get("/items/:id", func(c *fiber.Ctx) error {
		_, ok := bind.Params[idParam](c)
		if !ok {
			return nil
		}
		return c.SendString("ok")
	})

	ok, err := app.Test(httptest.NewRequest("GET", "/items/42", nil))
	require.NoError(t, err)
	require.Equal(t, 200, ok.StatusCode)

	bad, err := app.Test(httptest.NewRequest("GET", "/items/abc", nil))
	require.NoError(t, err)
	require.Equal(t, 400, bad.StatusCode) // ParamsParser fails to parse "abc" into int
	require.Contains(t, bodyOf(t, bad), `"status":"error"`)
}
