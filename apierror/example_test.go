// Copyright 2026 Rahmad Afandi. MIT License.

package apierror_test

import (
	"fmt"
	"io"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/apierror"
)

// Return a typed error from a handler; the ErrorHandler renders it as JSON.
func ExampleHandler() {
	app := fiber.New(fiber.Config{ErrorHandler: apierror.Handler})
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		return apierror.NotFound("user not found").WithCode("user_not_found")
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/users/9", nil))
	body, _ := io.ReadAll(resp.Body)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	// Output:
	// 404
	// {"code":404,"message":"user not found","error":"user_not_found","status":"error"}
}
