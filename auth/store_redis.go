// Copyright 2026 Rahmad Afandi. MIT License.

package auth

import (
	"context"
	"time"

	fhredis "github.com/rahmadafandi/fibr/redis"
	redislib "github.com/redis/go-redis/v9"
)

// RedisStore is a redis-backed TokenStore. It uses the underlying go-redis
// client for native key operations (string set with expiry, existence checks,
// delete).
type RedisStore struct {
	r      *fhredis.Redis
	prefix string
}

// StoreOption configures a RedisStore.
type StoreOption func(*RedisStore)

// WithStorePrefix sets the redis key prefix (default "auth:").
func WithStorePrefix(prefix string) StoreOption {
	return func(s *RedisStore) {
		if prefix != "" {
			s.prefix = prefix
		}
	}
}

// NewRedisStore creates a redis-backed token store on top of the redis package.
func NewRedisStore(r *fhredis.Redis, opts ...StoreOption) *RedisStore {
	s := &RedisStore{r: r, prefix: "auth:"}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *RedisStore) blockKey(jti string) string  { return s.prefix + "block:" + jti }
func (s *RedisStore) familyKey(fid string) string { return s.prefix + "family:" + fid }

// Block marks jti revoked until ttl elapses. A non-positive ttl is a no-op
// (an already-expired token needs no blocking) and avoids creating a key with
// no expiry, matching MemoryStore's behavior.
func (s *RedisStore) Block(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return s.r.Client.Set(ctx, s.blockKey(jti), "1", ttl).Err()
}

// IsBlocked reports whether jti is currently revoked.
func (s *RedisStore) IsBlocked(ctx context.Context, jti string) (bool, error) {
	n, err := s.r.Client.Exists(ctx, s.blockKey(jti)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SetFamily records jti as the active refresh token for family fid until ttl.
// A non-positive ttl is a no-op (avoids creating a key with no expiry).
func (s *RedisStore) SetFamily(ctx context.Context, fid, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return s.r.Client.Set(ctx, s.familyKey(fid), jti, ttl).Err()
}

// Family returns the active refresh jti for fid.
func (s *RedisStore) Family(ctx context.Context, fid string) (string, bool, error) {
	v, err := s.r.Client.Get(ctx, s.familyKey(fid)).Result()
	if err == redislib.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// RevokeFamily removes the family record, invalidating all its tokens.
func (s *RedisStore) RevokeFamily(ctx context.Context, fid string) error {
	return s.r.Client.Del(ctx, s.familyKey(fid)).Err()
}
