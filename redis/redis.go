// Copyright 2026 Rahmad Afandi. MIT License.

package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type (
	// Client is re-exported from go-redis so callers need not import that package
	// directly.
	Client = redis.Client
)

// Redis wraps a go-redis client with JSON (de)serialization helpers.
type Redis struct {
	Client *Client
}

// New creates a new Redis wrapper.
func New(client *Client) *Redis {
	return &Redis{Client: client}
}

// Set JSON-encodes value and stores it under key with the given expiration.
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	p, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, key, p, expiration).Err()
}

// Get reads key and JSON-decodes it into dest. Returns an error on cache miss.
func (r *Redis) Get(ctx context.Context, key string, dest interface{}) error {
	p, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(p, dest)
}

// Remember returns the cached value for key, or runs loader on a cache miss,
// stores its result with ttl, and returns it. A failure to store the loaded
// value does not fail the call.
func Remember[T any](ctx context.Context, r *Redis, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	var out T
	if err := r.Get(ctx, key, &out); err == nil {
		return out, nil
	}

	val, err := loader()
	if err != nil {
		var zero T
		return zero, err
	}

	// Best-effort cache write; ignore store errors so the loaded value is still returned.
	_ = r.Set(ctx, key, val, ttl)
	return val, nil
}

// ParseRedisOptions parses a redis:// URL into options.
func ParseRedisOptions(redisURL string) (*redis.Options, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid url: %w", err)
	}
	return opt, nil
}
