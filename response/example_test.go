// Copyright 2026 Rahmad Afandi. MIT License.

package response_test

import (
	"fmt"
	"io"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/response"
)

// SendSuccess wraps the payload in the standard JSON envelope
// ({code, message, data, status}) and writes it with a 200 status.
func ExampleSendSuccess() {
	app := fiber.New()
	app.Get("/ping", func(c *fiber.Ctx) error {
		return response.SendSuccess(c, fiber.Map{"pong": true}, "ok")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/ping", nil))
	body, _ := io.ReadAll(resp.Body)

	fmt.Println("status:", resp.StatusCode)
	fmt.Println("body:", string(body))
	// Output:
	// status: 200
	// body: {"code":200,"message":"ok","data":{"pong":true},"status":"success"}
}
