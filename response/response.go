// Copyright 2026 Rahmad Afandi. MIT License.

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
