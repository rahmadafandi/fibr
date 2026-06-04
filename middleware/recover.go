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
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/context"
	"github.com/rahmadafandi/fiber-helpers/logger"
	"github.com/rahmadafandi/fiber-helpers/response"
)

// Recover is a middleware that recovers from panics and logs the error.
func Recover(logger *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("%v", r)
				logger.Error(err, string(debug.Stack()), "request_id", context.GetRequestID(c))
				c.Status(fiber.StatusInternalServerError).JSON(response.Response{
					Code:    fiber.StatusInternalServerError,
					Message: "Internal Server Error",
					Status:  "error",
				})
			}
		}()
		return c.Next()
	}
}
