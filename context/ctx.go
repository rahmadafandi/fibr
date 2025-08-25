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

package context

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

func GetContext(c *fiber.Ctx) context.Context {
	if ctx, ok := c.Locals("ctx").(context.Context); ok {
		return ctx
	}
	return context.Background()
}

func GetRequestID(c *fiber.Ctx) string {
	if requestID, ok := c.Locals("requestid").(string); ok {
		return requestID
	}
	return ""
}

func CustomContext(c *fiber.Ctx, key string, value ...any) string {
	if len(value) == 0 {
		if v, ok := c.Locals(key).(string); ok {
			return v
		}
		return ""
	}
	if v, ok := c.Locals(key, value[0]).(string); ok {
		return v
	}
	return ""
}