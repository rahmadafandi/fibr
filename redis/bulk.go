// Copyright 2026 Rahmad Afandi. MIT License.

package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// MSet JSON-encodes and stores each pair. When ttl > 0 it is applied to every
// key. Executed in a single pipeline round-trip.
func (r *Redis) MSet(ctx context.Context, pairs map[string]any, ttl time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}
	pipe := r.Client.Pipeline()
	for k, v := range pairs {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("redis: encode %s: %w", k, err)
		}
		pipe.Set(ctx, k, b, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// MGet returns the raw stored value for each key that exists, keyed by key.
// Missing keys are omitted. Callers unmarshal the raw values themselves.
func (r *Redis) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	vals, err := r.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(keys))
	for i, v := range vals {
		if s, ok := v.(string); ok {
			out[keys[i]] = s
		}
	}
	return out, nil
}

// Incr atomically increments the integer value of key by one and returns the new value.
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.Client.Incr(ctx, key).Result()
}

// Decr atomically decrements the integer value of key by one and returns the new value.
func (r *Redis) Decr(ctx context.Context, key string) (int64, error) {
	return r.Client.Decr(ctx, key).Result()
}

// SetNX JSON-encodes value and sets it only if key does not already exist.
// It reports whether the key was set.
func (r *Redis) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("redis: encode: %w", err)
	}
	return r.Client.SetNX(ctx, key, b, ttl).Result()
}

// GetSet JSON-encodes value, stores it under key, and returns the previous raw
// value. The returned string is empty (with a redis.Nil error) when key did not exist.
func (r *Redis) GetSet(ctx context.Context, key string, value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("redis: encode: %w", err)
	}
	return r.Client.SetArgs(ctx, key, b, redis.SetArgs{Get: true}).Result()
}
