// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/bootstrap"
	"github.com/rahmadafandi/fibr/config"
	"github.com/rahmadafandi/fibr/database"
	"github.com/rahmadafandi/fibr/health"
	"github.com/rahmadafandi/fibr/response"
)

func main() {
	type Config struct {
		DatabaseURL string `mapstructure:"DATABASE_URL" default:"file::memory:?cache=shared"`
	}

	var cfg Config
	if err := config.LoadConfig(&cfg); err != nil {
		panic(err)
	}

	db, err := database.NewBun(cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}

	app := bootstrap.New(bootstrap.Options{
		DB:           db,
		EnableCORS:   true,
		RateLimit:    100,
		HealthChecks: []health.NamedCheck{health.PingBun(db)},
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return response.SendSuccess(c, "Hello, World!", "Welcome")
	})

	fmt.Println("Server listening on :3000")
	if err := app.Run(":3000"); err != nil {
		panic(err)
	}
}
