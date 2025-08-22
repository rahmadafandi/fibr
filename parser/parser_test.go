package parser

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	app := fiber.New()

	type TestStruct struct {
		Name string `json:"name"`
	}

	t.Run("ParseBody", func(t *testing.T) {
		app.Post("/body", func(c *fiber.Ctx) error {
			body, err := ParseBody[TestStruct](c)
			assert.NoError(t, err)
			assert.Equal(t, "test", body.Name)
			return nil
		})

		req := httptest.NewRequest("POST", "/body", bytes.NewBuffer([]byte(`{"name":"test"}`)))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	})

	t.Run("ParseQuery", func(t *testing.T) {
		app.Get("/query", func(c *fiber.Ctx) error {
			query, err := ParseQuery[TestStruct](c)
			assert.NoError(t, err)
			assert.Equal(t, "test", query.Name)
			return nil
		})

		req := httptest.NewRequest("GET", "/query?name=test", nil)
		app.Test(req)
	})

	t.Run("ParseParams", func(t *testing.T) {
		app.Get("/params/:name", func(c *fiber.Ctx) error {
			params, err := ParseParams[TestStruct](c)
			assert.NoError(t, err)
			assert.Equal(t, "test", params.Name)
			return nil
		})

		req := httptest.NewRequest("GET", "/params/test", nil)
		app.Test(req)
	})
}
