// Copyright 2026 Rahmad Afandi. MIT License.

package fibrtest_test

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/rahmadafandi/fibr/fibrtest"
)

// Drive a Fiber app under test with a few lines: build the request, assert the
// status, and decode the JSON response.
func ExampleClient() {
	app := fiber.New()
	app.Post("/greet", func(c *fiber.Ctx) error {
		var in struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&in); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"hello": in.Name})
	})

	// In a real test this is your *testing.T.
	t := &testing.T{}
	c := fibrtest.New(t, app)

	var out struct {
		Hello string `json:"hello"`
	}
	c.Post("/greet", map[string]string{"name": "ada"}).ExpectStatus(201).JSON(&out)

	fmt.Println(out.Hello)
	// Output: ada
}
