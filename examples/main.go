package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	jwtMain "github.com/golang-jwt/jwt/v5"
	"github.com/rahmadafandi/fiber-helpers/config"
	"github.com/rahmadafandi/fiber-helpers/jwt"
	"github.com/rahmadafandi/fiber-helpers/logger"
	"github.com/rahmadafandi/fiber-helpers/middleware"
	"github.com/rahmadafandi/fiber-helpers/parser"
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

	cfg, err := config.LoadConfig[Config](".")
	if err != nil {
		panic(err)
	}

	// Create logger
	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	log := logger.New(os.Stdout, logLevel)

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

		body, err := parser.ParseBody[LoginRequest](c)
		if err != nil {
			return response.SendError(c, nil, err.Error(), fiber.StatusBadRequest)
		}

		if errs := validator.ValidateStruct(body); len(errs) > 0 {
			return response.SendError(c, errs, validator.ErrorsToString(errs), fiber.StatusBadRequest)
		}

		// In a real app, you would check the password here

		claims := jwtMain.MapClaims{
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
		claims, err := jwt.GetClaims(c)
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
