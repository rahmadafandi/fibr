// Copyright 2026 Rahmad Afandi. MIT License.

package outbox

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// redisPublisher publishes raw event payload bytes to Redis channels.
type redisPublisher struct {
	client redis.UniversalClient
}

// NewRedisPublisher returns a Publisher that publishes the raw payload bytes on
// the topic channel via client.Publish, without re-encoding. The channel
// message is therefore exactly the stored JSON, so redis.Subscribe[T] on the
// consumer side decodes it directly.
func NewRedisPublisher(client redis.UniversalClient) Publisher {
	return &redisPublisher{client: client}
}

func (p *redisPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	return p.client.Publish(ctx, topic, payload).Err()
}
