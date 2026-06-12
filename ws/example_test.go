// Copyright 2026 Rahmad Afandi. MIT License.

package ws_test

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/ws"
)

type message struct {
	Text string `json:"text"`
}

func ExampleHub() {
	hub := ws.NewHub[message]()
	app := fiber.New()
	app.Get("/ws/:room", hub.Handle(ws.Handler[message]{
		OnConnect: func(c *ws.Conn[message]) error { c.Join(c.Params("room")); return nil },
		OnMessage: func(c *ws.Conn[message], m message) { hub.ToRoom(c.Params("room"), m) },
	}))
	_ = app
}
