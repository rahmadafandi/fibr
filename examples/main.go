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

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fiber-helpers/bootstrap"
	"github.com/rahmadafandi/fiber-helpers/config"
	"github.com/rahmadafandi/fiber-helpers/database"
	"github.com/rahmadafandi/fiber-helpers/health"
	"github.com/rahmadafandi/fiber-helpers/response"
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
