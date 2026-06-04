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

import "github.com/gofiber/fiber/v2"

// Response is the standard JSON envelope returned by all API handlers.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Status  string      `json:"status"`
}

// SendSuccess writes a JSON success response. The status code defaults to 200
// unless an optional code argument is provided.
func SendSuccess(c *fiber.Ctx, data interface{}, message string, code ...int) error {
	statusCode := fiber.StatusOK
	if len(code) > 0 {
		statusCode = code[0]
	}
	return c.Status(statusCode).JSON(&Response{
		Code:    statusCode,
		Message: message,
		Data:    data,
		Status:  "success",
	})
}

// SendError writes a JSON error response. The status code defaults to 400
// unless an optional code argument is provided.
func SendError(c *fiber.Ctx, data interface{}, message string, code ...int) error {
	statusCode := fiber.StatusBadRequest
	if len(code) > 0 {
		statusCode = code[0]
	}
	return c.Status(statusCode).JSON(&Response{
		Code:    statusCode,
		Message: message,
		Data:    data,
		Status:  "error",
	})
}
