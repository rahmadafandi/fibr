package common

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/response"
)

func HandleError(c *fiber.Ctx, err error, msg string) error {
	if err != nil {
		log.Printf("Error: %s, Details: %v", msg, err)
		return response.SendError(c, nil, msg)
	}
	return nil
}
