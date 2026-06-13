// Copyright 2026 Rahmad Afandi. MIT License.

// Package ws provides a typed WebSocket hub with rooms and JSON broadcast on
// top of github.com/gofiber/contrib/websocket. A hub is in-memory by default;
// WithRedis bridges broadcasts across replicas via Redis pub/sub.
package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rahmadafandi/fibr/redis"
)

// Hub tracks connections and rooms for messages of type T.
type Hub[T any] struct {
	cfg   hubConfig
	mu    sync.RWMutex
	conns map[*Conn[T]]struct{}
	rooms map[string]map[*Conn[T]]struct{}
	bp    *backplane[T] // nil when in-memory
}

type hubConfig struct {
	rds     *redis.Redis
	channel string
	ping    time.Duration
	sendBuf int
}

// Option configures a Hub.
type Option func(*hubConfig)

// WithRedis enables a Redis backplane so Broadcast/ToRoom reach clients on all
// replicas. channel is the pub/sub channel shared by every instance.
func WithRedis(r *redis.Redis, channel string) Option {
	return func(c *hubConfig) { c.rds = r; c.channel = channel }
}

// WithPingInterval sets the keepalive ping interval (default 30s; 0 disables).
func WithPingInterval(d time.Duration) Option { return func(c *hubConfig) { c.ping = d } }

// WithSendBuffer sets the per-connection outbound buffer (default 16).
func WithSendBuffer(n int) Option {
	return func(c *hubConfig) {
		if n > 0 {
			c.sendBuf = n
		}
	}
}

// NewHub creates a hub. With WithRedis it starts a backplane subscription.
func NewHub[T any](opts ...Option) *Hub[T] {
	cfg := hubConfig{ping: 30 * time.Second, sendBuf: 16}
	for _, o := range opts {
		o(&cfg)
	}
	h := &Hub[T]{
		cfg:   cfg,
		conns: map[*Conn[T]]struct{}{},
		rooms: map[string]map[*Conn[T]]struct{}{},
	}
	if cfg.rds != nil {
		h.bp = newBackplane(h)
	}
	return h
}

// Handler holds per-connection lifecycle callbacks (all optional).
type Handler[T any] struct {
	OnConnect    func(c *Conn[T]) error // return non-nil to reject the connection
	OnMessage    func(c *Conn[T], msg T)
	OnDisconnect func(c *Conn[T])
}

// Handle returns a Fiber handler that upgrades the request and runs the
// connection lifecycle. Mount it on a GET route.
func (h *Hub[T]) Handle(cfg Handler[T]) fiber.Handler {
	wsh := websocket.New(func(wc *websocket.Conn) {
		c := &Conn[T]{
			ws:    wc,
			hub:   h,
			send:  make(chan []byte, h.cfg.sendBuf),
			done:  make(chan struct{}),
			rooms: map[string]struct{}{},
		}
		if cfg.OnConnect != nil {
			if err := cfg.OnConnect(c); err != nil {
				return
			}
		}
		h.register(c)
		go c.writePump(h.cfg.ping)
		c.readPump(cfg, h.cfg.ping)
		if cfg.OnDisconnect != nil {
			cfg.OnDisconnect(c)
		}
		h.unregister(c)
		c.Close()
		// Wait for the write pump to stop before returning: gofiber/contrib
		// recycles the *websocket.Conn once this handler returns, which would
		// race with the write pump still touching it.
		<-c.done
	})
	return func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		return wsh(c)
	}
}

func (h *Hub[T]) register(c *Conn[T]) {
	h.mu.Lock()
	h.conns[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub[T]) unregister(c *Conn[T]) {
	h.mu.Lock()
	delete(h.conns, c)
	for room, set := range h.rooms {
		delete(set, c)
		if len(set) == 0 {
			delete(h.rooms, room)
		}
	}
	h.mu.Unlock()
}

func (h *Hub[T]) addToRoom(room string, c *Conn[T]) {
	h.mu.Lock()
	set := h.rooms[room]
	if set == nil {
		set = map[*Conn[T]]struct{}{}
		h.rooms[room] = set
	}
	set[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub[T]) removeFromRoom(room string, c *Conn[T]) {
	h.mu.Lock()
	if set := h.rooms[room]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.rooms, room)
		}
	}
	h.mu.Unlock()
}

// Broadcast sends msg to every connection (all replicas if a backplane is set).
func (h *Hub[T]) Broadcast(msg T) {
	if h.bp != nil {
		h.bp.publish("", msg)
		return
	}
	h.localBroadcast(msg)
}

// ToRoom sends msg to members of room (all replicas if a backplane is set).
func (h *Hub[T]) ToRoom(room string, msg T) {
	if h.bp != nil {
		h.bp.publish(room, msg)
		return
	}
	h.localToRoom(room, msg)
}

func (h *Hub[T]) localBroadcast(msg T) {
	b, err := marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	targets := make([]*Conn[T], 0, len(h.conns))
	for c := range h.conns {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		_ = c.enqueue(b)
	}
}

func (h *Hub[T]) localToRoom(room string, msg T) {
	b, err := marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	set := h.rooms[room]
	targets := make([]*Conn[T], 0, len(set))
	for c := range set {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		_ = c.enqueue(b)
	}
}

// Count returns the number of local connections.
func (h *Hub[T]) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

// BackplaneErr reports a Redis backplane subscription failure, or nil if the
// backplane is healthy or not configured. A hub with a failed backplane still
// delivers to its local connections but does not fan out across replicas.
func (h *Hub[T]) BackplaneErr() error {
	if h.bp == nil {
		return nil
	}
	return h.bp.err
}

// Close stops the backplane (if any) and closes all local connections.
func (h *Hub[T]) Close() error {
	if h.bp != nil {
		h.bp.close()
	}
	h.mu.Lock()
	conns := make([]*Conn[T], 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		c.Close()
	}
	return nil
}

func marshal[T any](msg T) ([]byte, error) { return json.Marshal(msg) }
