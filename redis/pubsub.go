// Copyright 2026 Rahmad Afandi. MIT License.

package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rahmadafandi/fibr/logger"
	"github.com/redis/go-redis/v9"
)

// Publish JSON-encodes value and publishes it to channel.
func (r *Redis) Publish(ctx context.Context, channel string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redis: encode publish payload: %w", err)
	}
	return r.Client.Publish(ctx, channel, b).Err()
}

// Subscription is an active subscription. Close stops the background goroutine
// and unsubscribes.
type Subscription struct {
	pubsub *redis.PubSub
}

// Close stops the subscription and releases its connection.
func (s *Subscription) Close() error { return s.pubsub.Close() }

// Subscribe subscribes to channel and invokes handler for each message (payload
// JSON-decoded into T) in a background goroutine, until ctx is cancelled or the
// returned Subscription is closed. Decode and handler errors are logged; a bad
// message is skipped without ending the subscription.
func Subscribe[T any](ctx context.Context, r *Redis, channel string, handler func(context.Context, T) error) (*Subscription, error) {
	ps := r.Client.Subscribe(ctx, channel)
	// Wait for the subscribe confirmation so a publish issued immediately after
	// Subscribe returns is not missed.
	if _, err := ps.Receive(ctx); err != nil {
		_ = ps.Close()
		return nil, fmt.Errorf("redis: subscribe %s: %w", channel, err)
	}

	log := logger.Default()
	go func() {
		ch := ps.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var v T
				if err := json.Unmarshal([]byte(msg.Payload), &v); err != nil {
					log.Error(err, "redis: decode pubsub message", "channel", channel)
					continue
				}
				if err := handler(ctx, v); err != nil {
					log.Error(err, "redis: pubsub handler", "channel", channel)
				}
			}
		}
	}()

	return &Subscription{pubsub: ps}, nil
}
