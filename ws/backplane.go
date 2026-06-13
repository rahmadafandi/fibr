// Copyright 2026 Rahmad Afandi. MIT License.

package ws

import (
	"context"

	"github.com/rahmadafandi/fibr/logger"
	"github.com/rahmadafandi/fibr/redis"
)

// backplane bridges hub broadcasts across replicas via Redis pub/sub. Every
// instance subscribes to the same channel; Broadcast/ToRoom publish an envelope
// that all instances (including the publisher) receive and fan out locally,
// giving a single delivery path.
type backplane[T any] struct {
	hub     *Hub[T]
	rds     *redis.Redis
	channel string
	sub     *redis.Subscription
	ctx     context.Context
	cancel  context.CancelFunc
	err     error // subscribe error, if the backplane failed to start
}

type envelope[T any] struct {
	Room string `json:"room,omitempty"`
	Data T      `json:"data"`
}

func newBackplane[T any](h *Hub[T]) *backplane[T] {
	ctx, cancel := context.WithCancel(context.Background())
	b := &backplane[T]{hub: h, rds: h.cfg.rds, channel: h.cfg.channel, ctx: ctx, cancel: cancel}
	sub, err := redis.Subscribe[envelope[T]](ctx, b.rds, b.channel, func(_ context.Context, env envelope[T]) error {
		if env.Room == "" {
			h.localBroadcast(env.Data)
		} else {
			h.localToRoom(env.Room, env.Data)
		}
		return nil
	})
	if err != nil {
		b.err = err
		logger.Default().Error(err, "ws: backplane subscribe", "channel", b.channel)
		return b
	}
	b.sub = sub
	return b
}

func (b *backplane[T]) publish(room string, msg T) {
	if err := b.rds.Publish(b.ctx, b.channel, envelope[T]{Room: room, Data: msg}); err != nil {
		logger.Default().Error(err, "ws: backplane publish", "channel", b.channel)
	}
}

func (b *backplane[T]) close() {
	b.cancel()
	if b.sub != nil {
		_ = b.sub.Close()
	}
}
