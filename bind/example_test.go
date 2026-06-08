// Copyright 2026 Rahmad Afandi. MIT License.

package bind_test

import (
	"fmt"
	"io"
	"net/http/httptest"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/bind"
)

// Body decodes and validates the request body in one call. On a validation
// failure it writes a 422 with per-field errors and returns ok=false, so the
// handler stops with `return nil`.
func ExampleBody() {
	type signupInput struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}

	app := fiber.New()
	app.Post("/signup", func(c *fiber.Ctx) error {
		in, ok := bind.Body[signupInput](c)
		if !ok {
			return nil // 400/422 already written
		}
		return c.SendString("welcome " + in.Email)
	})

	req := httptest.NewRequest("POST", "/signup", strings.NewReader(`{"email":"bad","password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)

	fmt.Println("status:", resp.StatusCode)
	fmt.Println("body:", string(body))
	// Output:
	// status: 422
	// body: {"code":422,"message":"validation failed","data":[{"field":"email","tag":"email","value":"bad"},{"field":"password","tag":"min","param":"8","value":"x"}],"status":"error"}
}
