// Copyright 2026 Rahmad Afandi. MIT License.

// Package redis wraps go-redis with cache-aside, a fiber.Storage adapter, pub/sub, and bulk/atomic helpers.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type (
	// Client is re-exported from go-redis so callers need not import that package
	// directly.
	Client = redis.Client
)

// Redis wraps a go-redis client with JSON (de)serialization helpers.
type Redis struct {
	Client *Client

	// group deduplicates concurrent Remember loads for the same key within this
	// process, preventing a cache stampede on a miss.
	group singleflight.Group
}

// New creates a new Redis wrapper.
func New(client *Client) *Redis {
	return &Redis{Client: client}
}

// Set JSON-encodes value and stores it under key with the given expiration.
func (r *Redis) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	p, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, key, p, expiration).Err()
}

// Get reads key and JSON-decodes it into dest. Returns an error on cache miss.
func (r *Redis) Get(ctx context.Context, key string, dest any) error {
	p, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(p, dest)
}

// Remember returns the cached value for key, or runs loader on a cache miss,
// stores its result with ttl, and returns it. A failure to store the loaded
// value does not fail the call. Concurrent misses for the same key within this
// process are deduplicated so loader runs once (cache-stampede protection).
func Remember[T any](ctx context.Context, r *Redis, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	var out T
	if err := r.Get(ctx, key, &out); err == nil {
		return out, nil
	}

	v, err, _ := r.group.Do(key, func() (any, error) {
		// Another flight may have populated the cache while we waited.
		var inner T
		if err := r.Get(ctx, key, &inner); err == nil {
			return inner, nil
		}
		val, err := loader()
		if err != nil {
			return nil, err
		}
		// Best-effort cache write; ignore store errors so the loaded value is still returned.
		_ = r.Set(ctx, key, val, ttl)
		return val, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return v.(T), nil
}

// Delete removes the given keys. It is a no-op (returns nil) when no keys are
// given.
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.Client.Del(ctx, keys...).Err()
}

// Exists reports whether key is present.
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.Client.Exists(ctx, key).Result()
	return n > 0, err
}

// Expire sets a time-to-live on key. The returned bool reports whether the key
// existed (and therefore whether the TTL was applied).
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.Client.Expire(ctx, key, ttl).Result()
}

// TTL returns the remaining lifetime of key. Following go-redis semantics, a
// missing key yields a negative duration (-2) and a key without an expiry
// yields -1.
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}

// ParseRedisOptions parses a redis:// URL into options.
func ParseRedisOptions(redisURL string) (*redis.Options, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid url: %w", err)
	}
	return opt, nil
}
