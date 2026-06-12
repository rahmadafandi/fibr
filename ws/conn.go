// Copyright 2026 Rahmad Afandi. MIT License.

package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/rahmadafandi/fibr/logger"
)

// Conn is one client connection carrying messages of type T.
type Conn[T any] struct {
	ws   *websocket.Conn
	hub  *Hub[T]
	send chan []byte
	done chan struct{} // closed when the write pump has fully stopped

	mu     sync.Mutex
	rooms  map[string]struct{}
	closed bool
	once   sync.Once
}

// Join adds the connection to a room for ToRoom fanout.
func (c *Conn[T]) Join(room string) {
	c.mu.Lock()
	c.rooms[room] = struct{}{}
	c.mu.Unlock()
	c.hub.addToRoom(room, c)
}

// Leave removes the connection from a room.
func (c *Conn[T]) Leave(room string) {
	c.mu.Lock()
	delete(c.rooms, room)
	c.mu.Unlock()
	c.hub.removeFromRoom(room, c)
}

// Rooms returns the rooms the connection currently belongs to.
func (c *Conn[T]) Rooms() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.rooms))
	for r := range c.rooms {
		out = append(out, r)
	}
	return out
}

// Params returns a route parameter captured at upgrade.
func (c *Conn[T]) Params(key string, def ...string) string { return c.ws.Params(key, def...) }

// Query returns a query-string value captured at upgrade.
func (c *Conn[T]) Query(key string, def ...string) string { return c.ws.Query(key, def...) }

// Locals returns a value stored on the request before upgrade.
func (c *Conn[T]) Locals(key string) any { return c.ws.Locals(key) }

// Send JSON-encodes msg and enqueues it to this connection.
func (c *Conn[T]) Send(msg T) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.enqueue(b)
}

func (c *Conn[T]) enqueue(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	select {
	case c.send <- b:
		return nil
	default:
		go c.Close() // slow client; closing takes the lock, so defer it
		return ErrSlowClient
	}
}

// Close shuts down the connection. Safe to call multiple times.
func (c *Conn[T]) Close() {
	c.once.Do(func() {
		c.mu.Lock()
		c.closed = true
		close(c.send)
		c.mu.Unlock()
	})
}

func (c *Conn[T]) writePump(ping time.Duration) {
	defer close(c.done)                 // runs last: signals the handler it is safe to return
	defer func() { _ = c.ws.Close() }() // runs first: close the underlying conn
	var tick <-chan time.Time
	if ping > 0 {
		t := time.NewTicker(ping)
		defer t.Stop()
		tick = t.C
	}
	for {
		select {
		case b, ok := <-c.send:
			if !ok {
				_ = c.ws.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		case <-tick:
			if err := c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				return
			}
		}
	}
}

func (c *Conn[T]) readPump(cfg Handler[T], ping time.Duration) {
	if ping > 0 {
		_ = c.ws.SetReadDeadline(time.Now().Add(ping * 2))
		c.ws.SetPongHandler(func(string) error {
			return c.ws.SetReadDeadline(time.Now().Add(ping * 2))
		})
	}
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		var msg T
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Default().Error(err, "ws: decode message")
			continue
		}
		if cfg.OnMessage != nil {
			cfg.OnMessage(c, msg)
		}
	}
}
