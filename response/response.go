package response

import "github.com/gofiber/fiber/v2"

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Status  string      `json:"status"`
}

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
