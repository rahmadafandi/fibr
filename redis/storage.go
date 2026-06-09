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

// Set stores val under key with the given expiration (0 means no expiry).
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

// Reset deletes every key under the prefix (not the whole database).
func (s *Storage) Reset() error {
	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, s.prefix+"*", 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.client.Del(ctx, keys...).Err()
}

// Close closes the underlying go-redis client. In generated apps the limiter
// storage uses its own client, so closing it does not affect other consumers.
func (s *Storage) Close() error { return s.client.Close() }
