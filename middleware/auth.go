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
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/jwt"
	"github.com/rahmadafandi/fiber-helpers/response"
)

// Auth is a middleware that protects routes with JWT authentication.
func Auth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.SendError(c, nil, "Missing or malformed JWT", fiber.StatusUnauthorized)
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return response.SendError(c, nil, "Missing or malformed JWT", fiber.StatusUnauthorized)
		}

		token, err := jwt.ValidateToken(parts[1], secret)
		if err != nil || !token.Valid {
			return response.SendError(c, nil, "Invalid or expired JWT", fiber.StatusUnauthorized)
		}

		c.Locals("user", token)

		return c.Next()
	}
}
