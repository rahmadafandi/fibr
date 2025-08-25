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