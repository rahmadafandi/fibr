// Copyright 2026 Rahmad Afandi. MIT License.

package sse_test

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/sse"
)

func ExampleHandler() {
	app := fiber.New()
	app.Get("/events", sse.Handler(func(c *fiber.Ctx, s *sse.Stream) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for n := 0; ; n++ {
			if err := s.Send("tick", map[string]int{"n": n}); err != nil {
				return // client disconnected
			}
			select {
			case <-c.Context().Done():
				return
			case <-ticker.C:
			}
		}
	}))
	_ = app
}
