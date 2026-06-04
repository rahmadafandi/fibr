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

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/config"
	"github.com/rahmadafandi/fiber-helpers/jwt"
	"github.com/rahmadafandi/fiber-helpers/logger"
	"github.com/rahmadafandi/fiber-helpers/middleware"
	"github.com/rahmadafandi/fiber-helpers/response"
	"github.com/rahmadafandi/fiber-helpers/validator"
	"github.com/rs/zerolog"
)

func main() {
	// Load config
	type Config struct {
		JWTSecret string `mapstructure:"JWT_SECRET"`
		LogLevel  string `mapstructure:"LOG_LEVEL"`
	}

	var cfg Config
	err := config.LoadConfig(&cfg)
	if err != nil {
		panic(err)
	}

	// Create logger
	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	log := logger.New(os.Stdout, logLevel)
	log.Info(fmt.Sprintf("Config: %+v", cfg))

	// Create fiber app
	app := fiber.New()

	// Middleware
	app.Use(middleware.Recover(log))
	app.Use(middleware.ContextMiddleware(10 * time.Second))
	app.Use(middleware.RequestLogger(log))

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return response.SendSuccess(c, "Hello, World!", "Welcome")
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		type LoginRequest struct {
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required"`
		}

		var body LoginRequest
		if err := c.BodyParser(&body); err != nil {
			return response.SendError(c, nil, err.Error(), fiber.StatusBadRequest)
		}

		if errs := validator.ValidateStruct(body); len(errs) > 0 {
			return response.SendError(c, errs, validator.ErrorsToString(errs), fiber.StatusBadRequest)
		}

		// In a real app, you would check the password here

		claims := jwt.MapClaims{
			"email": body.Email,
		}

		token, err := jwt.GenerateToken(claims, cfg.JWTSecret)
		if err != nil {
			return response.SendError(c, nil, err.Error(), fiber.StatusInternalServerError)
		}

		return response.SendSuccess(c, fiber.Map{"token": token}, "Login successful")
	})

	// Protected route
	app.Get("/protected", middleware.Auth(cfg.JWTSecret), func(c *fiber.Ctx) error {
		claims, err := jwt.ExtractClaimsFromJwt(c.Locals("user").(*jwt.Token))
		if err != nil {
			return response.SendError(c, nil, err.Error(), fiber.StatusInternalServerError)
		}

		return response.SendSuccess(c, claims, "Welcome")
	})

	// Start server
	port := 3000
	fmt.Printf("Server listening on port %d\n", port)
	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatal(err, "Failed to start server")
	}
}
