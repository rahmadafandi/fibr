// Copyright 2026 Rahmad Afandi. MIT License.

package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Storage adapts a go-redis client to the fiber.Storage interface (used by the
// rate limiter and other Fiber middleware). Keys are namespaced with a prefix so
// Reset is scoped and entries never collide with application cache keys.
type Storage struct {
	client *Client
	prefix string
}

// StorageOption configures a Storage.
type StorageOption func(*Storage)

// WithPrefix sets the key namespace (default "fiber:storage:").
func WithPrefix(prefix string) StorageOption {
	return func(s *Storage) { s.prefix = prefix }
}

// NewStorage adapts client to fiber.Storage.
func NewStorage(client *Client, opts ...StorageOption) *Storage {
	s := &Storage{client: client, prefix: "fiber:storage:"}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Get returns the value for key, or (nil, nil) if it is missing.
func (s *Storage) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, nil
	}
	b, err := s.client.Get(context.Background(), s.prefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Set stores val under key with the given expiration (0 means no expiry). An
// empty key or value is a no-op, matching fiber's storage adapters (nil and
// []byte{} are treated alike).
func (s *Storage) Set(key string, val []byte, exp time.Duration) error {
	if key == "" || len(val) == 0 {
		return nil
	}
	return s.client.Set(context.Background(), s.prefix+key, val, exp).Err()
}

// Delete removes key.
func (s *Storage) Delete(key string) error {
	if key == "" {
		return nil
	}
	return s.client.Del(context.Background(), s.prefix+key).Err()
}

// Reset deletes every key under the prefix (not the whole database). It deletes
// in batches so a large key space does not build one giant DEL.
func (s *Storage) Reset() error {
	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, s.prefix+"*", 0).Iterator()
	batch := make([]string, 0, 500)
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= 500 {
			if err := s.client.Del(ctx, batch...).Err(); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(batch) > 0 {
		return s.client.Del(ctx, batch...).Err()
	}
	return nil
}

// Close closes the underlying go-redis client. Call it only if this Storage owns
// the client; do not close a client shared with other consumers here.
func (s *Storage) Close() error { return s.client.Close() }
