package response

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	app := fiber.New()

	t.Run("SendSuccess", func(t *testing.T) {
		app.Get("/success", func(c *fiber.Ctx) error {
			return SendSuccess(c, "test", "test message")
		})

		req := httptest.NewRequest("GET", "/success", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("SendError", func(t *testing.T) {
		app.Get("/error", func(c *fiber.Ctx) error {
			return SendError(c, nil, "test error", fiber.StatusInternalServerError)
		})

		req := httptest.NewRequest("GET", "/error", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}
