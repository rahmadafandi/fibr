// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rahmadafandi/fiber-helpers/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	app := fiber.New()
	log := logger.New(&bytes.Buffer{}, zerolog.InfoLevel)

	t.Run("Recover", func(t *testing.T) {
		app.Get("/recover", Recover(log), func(c *fiber.Ctx) error {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/recover", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Auth", func(t *testing.T) {
		secret := "secret"
		app.Get("/auth", Auth(secret), func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		// Valid token
		claims := jwt.MapClaims{
			"name": "test",
			"exp":  time.Now().Add(time.Hour * 24).Unix(),
		}
		token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))

		req := httptest.NewRequest("GET", "/auth", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Invalid token
		req = httptest.NewRequest("GET", "/auth", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		resp, _ = app.Test(req)

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("RequestLogger", func(t *testing.T) {
		app.Get("/logger", RequestLogger(log), func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/logger", nil)
		app.Test(req)
	})
}

func TestContextMiddlewareSetsRequestIDHeader(t *testing.T) {
	app := fiber.New()
	app.Use(ContextMiddleware(time.Second))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}
